import numpy as np

from inference.backends.base import ForecastBackend


class StubPipeline:
    """StubPipeline is a stand-in pipeline that records nothing and returns fixed outputs."""

    loaded = False


class StubBackend(ForecastBackend[StubPipeline]):
    """StubBackend implements the abstract backend hooks with trivial outputs for lifecycle tests."""

    def _load(self) -> StubPipeline:
        """_load returns a fresh stub pipeline."""
        pipeline = StubPipeline()
        pipeline.loaded = True
        return pipeline

    def _predict_loaded(
        self,
        pipeline: StubPipeline,
        target: np.ndarray,
        past_covariates: dict[str, np.ndarray],
        future_covariates: dict[str, np.ndarray],
        prediction_length: int,
    ) -> tuple[np.ndarray, np.ndarray]:
        """_predict_loaded returns fixed quantiles and mean shaped per the backend contract."""
        variances = target.shape[0]
        quantiles = np.zeros((variances, prediction_length, 3), dtype=np.float32)
        mean = np.zeros((variances, prediction_length), dtype=np.float32)
        return quantiles, mean


def test_backend_lazy_loading() -> None:
    """Test_backend_lazy_loading verifies the pipeline loads on first use only."""
    backend = StubBackend(model_name="stub")
    assert not backend.is_loaded()

    pipeline = backend.get_pipeline()
    assert backend.is_loaded()
    assert pipeline.loaded

    assert backend.get_pipeline() is pipeline


def test_backend_predict_contract() -> None:
    """Test_backend_predict_contract verifies predict shapes and delegated loading."""
    backend = StubBackend(model_name="stub")
    target = np.zeros((2, 16), dtype=np.float32)

    quantiles, mean = backend.predict(
        target=target,
        past_covariates={},
        future_covariates={},
        prediction_length=8,
    )

    assert backend.is_loaded()
    assert quantiles.shape == (2, 8, 3)
    assert mean.shape == (2, 8)


def test_backend_idle_eviction() -> None:
    """Test_backend_idle_eviction verifies eviction only after the idle timeout expires."""
    backend = StubBackend(model_name="stub", idle_timeout_seconds=3600.0)
    backend.get_pipeline()
    assert backend.is_loaded()

    backend._last_access_time = 1000.0
    assert not backend.maybe_unload_idle(now=1500.0)
    assert backend.is_loaded()

    assert backend.maybe_unload_idle(now=5000.0)
    assert not backend.is_loaded()


def test_backend_preloaded_pipeline_skips_load() -> None:
    """Test_backend_preloaded_pipeline_skips_load verifies an injected pipeline is reused."""
    pipeline = StubPipeline()
    pipeline.loaded = True
    backend = StubBackend(model_name="stub", pipeline=pipeline)

    assert backend.is_loaded()
    assert backend.get_pipeline() is pipeline
