import numpy as np
from timestral import TimestralForecaster

from inference.backends.base import QUANTILE_LEVELS, ForecastBackend

# Maximum allowed distance between a requested level and the closest model knot.
_LEVEL_MATCH_TOLERANCE = 1e-3


def _level_index(levels: tuple[float, ...], level: float) -> int:
    """_level_index finds the closest model quantile knot to the requested level."""
    distances = [abs(available - level) for available in levels]
    index = int(np.argmin(distances))
    if distances[index] > _LEVEL_MATCH_TOLERANCE:
        raise ValueError(f"timestral checkpoint does not provide quantile level {level}: {levels}")
    return index


def _stack_known_covariates(
    past_covariates: dict[str, np.ndarray],
    future_covariates: dict[str, np.ndarray],
) -> np.ndarray | None:
    """_stack_known_covariates concatenates past and future covariate rows into one known-future tensor.

    The result has shape (1, C, T + H). Returns None when no covariates are supplied.
    """
    if not past_covariates:
        return None

    past_keys = list(past_covariates.keys())
    missing = [key for key in past_keys if key not in future_covariates]
    if missing:
        raise ValueError(f"future covariates are missing keys present in past covariates: {missing}")

    rows = [np.concatenate([past_covariates[key], future_covariates[key]], axis=0) for key in past_keys]
    stacked = np.stack(rows, axis=0).astype(np.float32)
    return stacked[None, ...]


class TimestralBackend(ForecastBackend[TimestralForecaster]):
    """TimestralBackend runs multivariate probabilistic forecasts with the Timestral forecaster."""

    def _load(self) -> TimestralForecaster:
        """_load initializes the Timestral forecaster on CPU."""
        return TimestralForecaster.from_pretrained(self.model_name, device="cpu")

    def _predict_loaded(
        self,
        pipeline: TimestralForecaster,
        target: np.ndarray,
        past_covariates: dict[str, np.ndarray],
        future_covariates: dict[str, np.ndarray],
        prediction_length: int,
    ) -> tuple[np.ndarray, np.ndarray]:
        """_predict_loaded runs one Timestral forecast and extracts quantiles and mean."""
        # (V, T) target becomes the single-series context (1, V, T)
        context = target[None, ...].astype(np.float32)
        known_covariates = _stack_known_covariates(past_covariates, future_covariates)

        forecast = pipeline.forecast(
            context,
            prediction_length=prediction_length,
            known_covariates=known_covariates,
        )

        # Quantiles arrive as (1, V, H, Q); select the contract levels into (V, H, 3)
        levels = forecast.quantile_levels
        selected = [
            forecast.quantiles[..., _level_index(levels, level)].detach().cpu().numpy() for level in QUANTILE_LEVELS
        ]
        quantiles_arr = np.stack(selected, axis=-1)[0]
        mean_arr = forecast.median[0].detach().cpu().numpy()
        return quantiles_arr, mean_arr
