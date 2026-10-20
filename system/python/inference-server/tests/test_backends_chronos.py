import numpy as np
import pytest

pytest.importorskip("chronos")

from inference.backends.chronos import Chronos2Backend
from tests.helpers import create_synthetic_chronos_pipeline


def test_chronos_predict_cpu_execution() -> None:
    """Test_chronos_predict_cpu_execution verifies multivariate forecasting on CPU."""
    synthetic_pipeline = create_synthetic_chronos_pipeline()
    backend = Chronos2Backend(model_name="synthetic-test", pipeline=synthetic_pipeline)

    history_length = 32
    prediction_length = 24
    target = np.random.randn(2, history_length).astype(np.float32)
    past_covariates = {
        "hour_sin": np.random.randn(history_length).astype(np.float32),
        "hour_cos": np.random.randn(history_length).astype(np.float32),
    }
    future_covariates = {
        "hour_sin": np.random.randn(prediction_length).astype(np.float32),
        "hour_cos": np.random.randn(prediction_length).astype(np.float32),
    }

    quantiles, mean = backend.predict(
        target=target,
        past_covariates=past_covariates,
        future_covariates=future_covariates,
        prediction_length=prediction_length,
    )

    assert quantiles.shape == (2, prediction_length, 3)
    assert mean.shape == (2, prediction_length)
    assert np.isfinite(quantiles).all()
    assert np.isfinite(mean).all()
