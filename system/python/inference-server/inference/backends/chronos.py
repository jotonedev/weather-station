import numpy as np
import torch
from chronos import Chronos2Pipeline

from inference.backends.base import QUANTILE_LEVELS, ForecastBackend


class Chronos2Backend(ForecastBackend[Chronos2Pipeline]):
    """Chronos2Backend runs multivariate probabilistic forecasts with the Chronos-2 pipeline."""

    def _load(self) -> Chronos2Pipeline:
        """_load initializes the Chronos-2 pipeline on CPU."""
        return Chronos2Pipeline.from_pretrained(
            self.model_name,
            device_map="cpu",
            torch_dtype=torch.float32,
        )

    def max_recommended_horizon(self) -> int | None:
        """max_recommended_horizon returns the library's recommended prediction length cap."""
        # The chronos library warns, but does not fail, when the requested prediction
        # length exceeds the checkpoint's trained horizon. Reading the loaded pipeline
        # keeps this truthful per checkpoint rather than hardcoding a value.
        return self.get_pipeline().model_prediction_length

    def _predict_loaded(
        self,
        pipeline: Chronos2Pipeline,
        target: np.ndarray,
        past_covariates: dict[str, np.ndarray],
        future_covariates: dict[str, np.ndarray],
        prediction_length: int,
    ) -> tuple[np.ndarray, np.ndarray]:
        """_predict_loaded runs one Chronos-2 forecast and extracts quantiles and mean."""
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
            quantile_levels=list(QUANTILE_LEVELS),
        )

        quantiles_arr = quantiles[0].detach().cpu().numpy()
        mean_arr = mean[0].detach().cpu().numpy()
        return quantiles_arr, mean_arr
