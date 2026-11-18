import threading
import time
from concurrent.futures import ThreadPoolExecutor

import numpy as np
import pytest

from inference.backends.base import QUANTILE_LEVELS, ForecastBackend


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
    ) -> tuple[np.ndarray, tuple[float, ...]]:
        """_predict_loaded returns fixed quantiles and levels shaped per the backend contract."""
        variates = target.shape[0]
        quantiles = np.zeros((variates, prediction_length, len(QUANTILE_LEVELS)), dtype=np.float32)
        return quantiles, QUANTILE_LEVELS


class CountingStubBackend(StubBackend):
    """CountingStubBackend counts _load invocations and optionally delays loading for concurrency tests."""

    def __init__(
        self,
        model_name: str,
        idle_timeout_seconds: float = 3600.0,
        pipeline: StubPipeline | None = None,
        load_delay_seconds: float = 0.0,
    ):
        super().__init__(
            model_name=model_name,
            idle_timeout_seconds=idle_timeout_seconds,
            pipeline=pipeline,
        )
        self.load_count = 0
        self.load_delay_seconds = load_delay_seconds

    def _load(self) -> StubPipeline:
        """_load increments load_count and returns a fresh stub pipeline."""
        self.load_count += 1
        if self.load_delay_seconds > 0.0:
            time.sleep(self.load_delay_seconds)
        return super()._load()


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

    quantiles, levels = backend.predict(
        target=target,
        past_covariates={},
        future_covariates={},
        prediction_length=8,
    )

    assert backend.is_loaded()
    assert quantiles.shape == (2, 8, len(QUANTILE_LEVELS))
    assert levels == QUANTILE_LEVELS


def test_backend_idle_eviction(monkeypatch: pytest.MonkeyPatch) -> None:
    """Test_backend_idle_eviction verifies exact timeout boundary eviction and reload."""
    monkeypatch.setattr("inference.backends.base.time.time", lambda: 1000.0)
    backend = StubBackend(model_name="stub", idle_timeout_seconds=60.0)
    first_pipeline = backend.get_pipeline()
    assert backend.is_loaded()

    assert not backend.maybe_unload_idle(now=1059.999)
    assert backend.is_loaded()

    assert backend.maybe_unload_idle(now=1060.0)
    assert not backend.is_loaded()
    assert backend.get_pipeline() is not first_pipeline


def test_backend_maybe_unload_idle_when_unloaded(monkeypatch: pytest.MonkeyPatch) -> None:
    """Test_backend_maybe_unload_idle_when_unloaded returns False when unloaded or already evicted."""
    monkeypatch.setattr("inference.backends.base.time.time", lambda: 1000.0)
    unloaded_backend = StubBackend(model_name="stub", idle_timeout_seconds=0.0)
    assert not unloaded_backend.is_loaded()
    assert not unloaded_backend.maybe_unload_idle(now=10_000.0)
    assert not unloaded_backend.is_loaded()

    evicted_backend = StubBackend(model_name="stub", idle_timeout_seconds=60.0)
    evicted_backend.get_pipeline()
    assert evicted_backend.maybe_unload_idle(now=1060.0)
    assert not evicted_backend.maybe_unload_idle(now=1061.0)


def test_backend_preloaded_pipeline_skips_load() -> None:
    """Test_backend_preloaded_pipeline_skips_load verifies an injected pipeline is reused without loading."""
    pipeline = StubPipeline()
    pipeline.loaded = True
    backend = CountingStubBackend(model_name="stub", pipeline=pipeline)

    assert backend.is_loaded()
    assert backend.get_pipeline() is pipeline
    assert backend.load_count == 0


def test_backend_concurrent_get_pipeline_loads_once() -> None:
    """Test_backend_concurrent_get_pipeline_loads_once verifies thread-safe single lazy initialization."""
    backend = CountingStubBackend(model_name="stub", load_delay_seconds=0.02)
    barrier = threading.Barrier(8)

    def load_worker() -> StubPipeline:
        barrier.wait()
        return backend.get_pipeline()

    with ThreadPoolExecutor(max_workers=8) as pool:
        results = list(pool.map(lambda _index: load_worker(), range(8)))

    assert backend.load_count == 1
    assert len({id(pipeline_obj) for pipeline_obj in results}) == 1
    assert backend.is_loaded()
