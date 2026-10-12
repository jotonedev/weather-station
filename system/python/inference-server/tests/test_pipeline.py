import numpy as np
from chronos import Chronos2Pipeline
from chronos.chronos2.config import Chronos2CoreConfig
from chronos.chronos2.model import Chronos2Model

from inference.pipeline import PipelineManager


def create_synthetic_pipeline() -> Chronos2Pipeline:
    """Create_synthetic_pipeline builds a lightweight in-memory Chronos2Pipeline for CPU testing."""
    core_config = Chronos2CoreConfig(d_model=32, d_kv=8, d_ff=64, num_layers=1, num_heads=2)
    all_quantiles = [0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9]
    core_config.chronos_config = {
        "context_length": 64,
        "output_patch_size": 8,
        "input_patch_size": 8,
        "input_patch_stride": 8,
        "quantiles": all_quantiles,
        "use_reg_token": True,
    }
    model = Chronos2Model(core_config)
    return Chronos2Pipeline(model)


def test_pipeline_lazy_loading_and_idle_eviction() -> None:
    """Test_pipeline_lazy_loading_and_idle_eviction verifies lazy loading and eviction when idle."""
    synthetic_pipeline = create_synthetic_pipeline()

    # Initialize manager without preloaded pipeline
    manager = PipelineManager(
        model_name="synthetic-test",
        idle_timeout_seconds=3600.0,
    )
    assert not manager.is_loaded()

    # Manually inject pipeline to simulate loading
    manager._pipeline = synthetic_pipeline
    manager._last_access_time = 1000.0
    assert manager.is_loaded()

    # Check eviction when not expired
    assert not manager.maybe_unload_idle(now=1500.0)
    assert manager.is_loaded()

    # Check eviction when expired
    assert manager.maybe_unload_idle(now=5000.0)
    assert not manager.is_loaded()


def test_pipeline_predict_cpu_execution() -> None:
    """Test_pipeline_predict_cpu_execution verifies multivariate forecasting on CPU."""
    synthetic_pipeline = create_synthetic_pipeline()
    manager = PipelineManager(
        model_name="synthetic-test",
        pipeline=synthetic_pipeline,
    )

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

    quantiles, mean = manager.predict(
        target=target,
        past_covariates=past_covariates,
        future_covariates=future_covariates,
        prediction_length=prediction_length,
    )

    assert quantiles.shape == (2, prediction_length, 3)
    assert mean.shape == (2, prediction_length)
