import gc
import threading
import time

import numpy as np
import torch
from chronos import Chronos2Pipeline


class PipelineManager:
    """PipelineManager handles lazy loading, CPU execution, and idle unloading for Chronos-2."""

    def __init__(
        self,
        model_name: str = "amazon/chronos-2",
        idle_timeout_seconds: float = 3600.0,
        pipeline: Chronos2Pipeline | None = None,
    ):
        """PipelineManager initializes pipeline state and idle management."""
        self.model_name = model_name
        self.idle_timeout_seconds = idle_timeout_seconds
        self._pipeline = pipeline
        self._last_access_time = time.time() if pipeline is not None else 0.0
        self._lock = threading.Lock()

    def get_pipeline(self) -> Chronos2Pipeline:
        """get_pipeline returns the loaded Chronos2Pipeline or initializes it on CPU."""
        with self._lock:
            if self._pipeline is None:
                self._pipeline = Chronos2Pipeline.from_pretrained(
                    self.model_name,
                    device_map="cpu",
                    torch_dtype=torch.float32,
                )
            self._last_access_time = time.time()
            return self._pipeline

    def is_loaded(self) -> bool:
        """is_loaded indicates whether the pipeline model is currently resident in memory."""
        with self._lock:
            return self._pipeline is not None

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
    ) -> tuple[np.ndarray, np.ndarray]:
        """Predict generates quantile and mean forecasts for multivariate target time series."""
        pipeline = self.get_pipeline()
        inputs = [
            {
                "target": target,
                "past_covariates": past_covariates,
                "future_covariates": future_covariates,
            }
        ]

        # Request 10th, 50th, and 90th percentiles for point forecast and confidence interval
        quantiles, mean = pipeline.predict_quantiles(
            inputs,
            prediction_length=prediction_length,
            quantile_levels=[0.1, 0.5, 0.9],
        )

        quantiles_arr = quantiles[0].detach().cpu().numpy()
        mean_arr = mean[0].detach().cpu().numpy()
        return quantiles_arr, mean_arr
