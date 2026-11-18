import numpy as np
from timestral import TimestralForecaster

from inference.backends.base import ForecastBackend, _split_covariates


class TimestralBackend(ForecastBackend[TimestralForecaster]):
    """TimestralBackend runs multivariate probabilistic forecasts with the Timestral forecaster."""

    def __init__(
        self,
        model_name: str,
        idle_timeout_seconds: float = 3600.0,
        pipeline: TimestralForecaster | None = None,
        symmetric_averaging: bool = False,
    ):
        """TimestralBackend stores model identity, idle timeout, and symmetric averaging flag."""
        super().__init__(
            model_name=model_name,
            idle_timeout_seconds=idle_timeout_seconds,
            pipeline=pipeline,
        )
        self.symmetric_averaging = symmetric_averaging

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
            clip_nonnegative=False,
            symmetric_averaging=self.symmetric_averaging,
        )

        quantiles_arr = forecast.quantiles[0].detach().cpu().numpy()
        levels = tuple(float(level) for level in forecast.quantile_levels)
        return quantiles_arr, levels
