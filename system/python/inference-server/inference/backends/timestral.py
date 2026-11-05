import numpy as np
from timestral import TimestralForecaster

from inference.backends.base import ForecastBackend


def _split_covariates(
    past_covariates: dict[str, np.ndarray],
    future_covariates: dict[str, np.ndarray],
) -> tuple[np.ndarray | None, np.ndarray | None]:
    """_split_covariates separates past-only covariates from known past+future covariates.

    Returns (past_tensor, known_tensor) with shapes (1, C_past, T) and (1, C_known, T + H),
    or None for either tensor when its corresponding channel set is empty.
    """
    missing_past = [key for key in future_covariates if key not in past_covariates]
    if missing_past:
        raise ValueError(f"past covariates are missing keys present in future covariates: {missing_past}")

    known_keys = [key for key in past_covariates if key in future_covariates]
    past_only_keys = [key for key in past_covariates if key not in future_covariates]

    past_tensor: np.ndarray | None = None
    if past_only_keys:
        past_rows = [np.asarray(past_covariates[key], dtype=np.float32) for key in past_only_keys]
        past_tensor = np.stack(past_rows, axis=0)[None, ...]

    known_tensor: np.ndarray | None = None
    if known_keys:
        known_rows = [
            np.concatenate([past_covariates[key], future_covariates[key]], axis=0).astype(np.float32)
            for key in known_keys
        ]
        known_tensor = np.stack(known_rows, axis=0)[None, ...]

    return past_tensor, known_tensor


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
    ) -> tuple[np.ndarray, tuple[float, ...]]:
        """_predict_loaded runs one Timestral forecast and returns all checkpoint quantiles and levels."""
        # (V, T) target becomes the single-series context (1, V, T)
        context = target[None, ...].astype(np.float32)
        past_tensor, known_tensor = _split_covariates(past_covariates, future_covariates)

        forecast = pipeline.forecast(
            context,
            prediction_length=prediction_length,
            past_covariates=past_tensor,
            known_covariates=known_tensor,
        )

        quantiles_arr = forecast.quantiles[0].detach().cpu().numpy()
        levels = tuple(float(level) for level in forecast.quantile_levels)
        return quantiles_arr, levels
