"""ForecastBackend defines the shared lifecycle and output contract for forecast model backends."""

from __future__ import annotations

import abc
import gc
import threading
import time

import numpy as np

# Default quantile levels for backends that require explicit level selection.
QUANTILE_LEVELS: tuple[float, ...] = (0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95)


class ForecastBackend[PipelineT](abc.ABC):
    """ForecastBackend provides lazy loading, CPU execution, and idle unloading for a forecast model."""

    def __init__(
        self,
        model_name: str,
        idle_timeout_seconds: float = 3600.0,
        pipeline: PipelineT | None = None,
    ):
        """ForecastBackend stores model identity, idle timeout, and an optional preloaded pipeline."""
        self.model_name = model_name
        self.idle_timeout_seconds = idle_timeout_seconds
        self._pipeline = pipeline
        self._last_access_time = time.time() if pipeline is not None else 0.0
        self._lock = threading.Lock()

    @abc.abstractmethod
    def _load(self) -> PipelineT:
        """_load initializes the model on CPU and returns it."""

    @abc.abstractmethod
    def _predict_loaded(
        self,
        pipeline: PipelineT,
        target: np.ndarray,
        past_covariates: dict[str, np.ndarray],
        future_covariates: dict[str, np.ndarray],
        prediction_length: int,
    ) -> tuple[np.ndarray, tuple[float, ...]]:
        """_predict_loaded runs one forecast on a loaded pipeline.

        It must return quantiles shaped (variates, horizon, num_quantiles) and
        the corresponding quantile levels tuple.
        """

    def get_pipeline(self) -> PipelineT:
        """get_pipeline returns the loaded pipeline or initializes it on CPU."""
        with self._lock:
            if self._pipeline is None:
                self._pipeline = self._load()
            self._last_access_time = time.time()
            return self._pipeline

    def is_loaded(self) -> bool:
        """is_loaded indicates whether the pipeline model is currently resident in memory."""
        with self._lock:
            return self._pipeline is not None

    def max_recommended_horizon(self) -> int | None:
        """max_recommended_horizon returns the recommended prediction horizon cap or None when unbounded."""
        return None

    def maybe_unload_idle(self, now: float | None = None) -> bool:
        """maybe_unload_idle unloads the model from memory if idle time exceeds timeout."""
        current_time = time.time() if now is None else now
        with self._lock:
            if self._pipeline is None:
                return False

            if current_time - self._last_access_time >= self.idle_timeout_seconds:
                self._pipeline = None
                gc.collect()
                return True

            return False

    def predict(
        self,
        target: np.ndarray,
        past_covariates: dict[str, np.ndarray],
        future_covariates: dict[str, np.ndarray],
        prediction_length: int,
    ) -> tuple[np.ndarray, tuple[float, ...]]:
        """Predict generates quantile forecasts for multivariate target time series.

        Returns quantiles shaped (variates, horizon, num_quantiles) and the
        corresponding quantile levels tuple.
        """
        pipeline = self.get_pipeline()
        return self._predict_loaded(
            pipeline=pipeline,
            target=target,
            past_covariates=past_covariates,
            future_covariates=future_covariates,
            prediction_length=prediction_length,
        )
