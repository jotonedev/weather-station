import logging
import threading
from collections.abc import Generator, Iterator
from concurrent import futures
from contextlib import contextmanager
from typing import cast

import grpc
import numpy as np
import pytest
from google.protobuf.timestamp_pb2 import Timestamp
from grpc_health.v1 import health_pb2, health_pb2_grpc
from weather.inference.v1 import inference_pb2, inference_pb2_grpc

from inference.backends.base import QUANTILE_LEVELS, ForecastBackend
from inference.config import Config
from inference.server import InferenceServer, InferenceServicer


class StubForecastBackend(ForecastBackend[object]):
    """StubForecastBackend implements ForecastBackend in memory without optional model dependencies."""

    def __init__(
        self,
        model_name: str = "stub-model",
        idle_timeout_seconds: float = 3600.0,
        pipeline: object | None = None,
        recommended_horizon: int | None = None,
        predict_error: Exception | None = None,
        on_unload: threading.Event | None = None,
    ):
        super().__init__(
            model_name=model_name,
            idle_timeout_seconds=idle_timeout_seconds,
            pipeline=pipeline,
        )
        self.recommended_horizon = recommended_horizon
        self.predict_error = predict_error
        self.on_unload = on_unload
        self.captured_calls: list[dict[str, object]] = []

    def _load(self) -> object:
        """_load returns an in-memory sentinel pipeline object."""
        return object()

    def max_recommended_horizon(self) -> int | None:
        """max_recommended_horizon returns the configured horizon recommendation."""
        return self.recommended_horizon

    def _predict_loaded(
        self,
        pipeline: object,
        target: np.ndarray,
        past_covariates: dict[str, np.ndarray],
        future_covariates: dict[str, np.ndarray],
        prediction_length: int,
    ) -> tuple[np.ndarray, tuple[float, ...]]:
        """_predict_loaded records invocation inputs and returns deterministic valid quantiles."""
        self.captured_calls.append(
            {
                "target": target.copy(),
                "past_covariates": dict(past_covariates),
                "future_covariates": dict(future_covariates),
                "prediction_length": prediction_length,
            }
        )
        if self.predict_error is not None:
            raise self.predict_error

        variates = target.shape[0]
        last_values = np.nan_to_num(target[:, -1:], nan=15.0).astype(np.float32)
        offsets = np.linspace(-2.0, 2.0, len(QUANTILE_LEVELS), dtype=np.float32)
        quantiles = last_values[:, :, None] + offsets[None, None, :]
        quantiles = np.broadcast_to(quantiles, (variates, prediction_length, len(QUANTILE_LEVELS))).copy()
        return quantiles, QUANTILE_LEVELS

    def maybe_unload_idle(self, now: float | None = None) -> bool:
        """maybe_unload_idle delegates eviction and notifies the test synchronization event."""
        evicted = super().maybe_unload_idle(now=now)
        if evicted and self.on_unload is not None:
            self.on_unload.set()
        return evicted


def _build_hourly_request(
    node_id: str = "station-alpha",
    horizon_hours: int = 24,
    latitude: float | None = 45.5,
    longitude: float | None = 9.2,
    elevation_m: float | None = 120.0,
    include_optional: bool = True,
) -> inference_pb2.PredictHourlyWeatherRequest:
    """_build_hourly_request constructs a valid hourly weather gRPC request."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    observations = []
    for step_index in range(32):
        timestamp = Timestamp()
        timestamp.FromSeconds(1700000000 - ((32 - step_index) * 3600))
        obs = inference_pb2.TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=22.0 + (step_index * 0.1),
            humidity_pct=60.0,
        )
        if include_optional:
            obs.pressure_hpa = 1013.0
            obs.rainfall_mm = 0.4
            obs.wind_direction_deg = 225.0
        observations.append(obs)

    request = inference_pb2.PredictHourlyWeatherRequest(
        node_id=node_id,
        issue_time=issue_time,
        horizon_hours=horizon_hours,
        history=observations,
    )
    if latitude is not None:
        request.latitude = latitude
    if longitude is not None:
        request.longitude = longitude
    if elevation_m is not None:
        request.elevation_m = elevation_m
    return request


def _build_daily_request(
    node_id: str = "station-beta",
    horizon_days: int = 10,
    latitude: float | None = 45.5,
    longitude: float | None = 9.2,
    elevation_m: float | None = 120.0,
) -> inference_pb2.PredictDailyWeatherRequest:
    """_build_daily_request constructs a valid daily weather gRPC request."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    observations = []
    for step_index in range(16):
        timestamp = Timestamp()
        timestamp.FromSeconds(1700000000 - ((16 - step_index) * 86400))
        obs = inference_pb2.TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=18.0,
            humidity_pct=65.0,
            min_temperature_c=12.0,
            max_temperature_c=22.0,
            min_humidity_pct=50.0,
            max_humidity_pct=80.0,
            pressure_hpa=1015.0,
        )
        observations.append(obs)

    request = inference_pb2.PredictDailyWeatherRequest(
        node_id=node_id,
        issue_time=issue_time,
        horizon_days=horizon_days,
        history=observations,
    )
    if latitude is not None:
        request.latitude = latitude
    if longitude is not None:
        request.longitude = longitude
    if elevation_m is not None:
        request.elevation_m = elevation_m
    return request


@contextmanager
def _start_servicer(
    backend: StubForecastBackend,
    config: Config | None = None,
) -> Generator[tuple[inference_pb2_grpc.InferenceServiceStub, StubForecastBackend]]:
    """_start_servicer starts an ephemeral in-process gRPC servicer and yields its stub and backend."""
    active_config = config or Config(host="127.0.0.1", port=0, model_name=backend.model_name, max_horizon_hours=240)
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    servicer = InferenceServicer(backend, active_config)
    inference_pb2_grpc.add_InferenceServiceServicer_to_server(servicer, server)
    port = server.add_insecure_port("127.0.0.1:0")
    server.start()

    channel = grpc.insecure_channel(
        f"127.0.0.1:{port}",
        options=[("grpc.enable_http_proxy", 0)],
    )
    stub = inference_pb2_grpc.InferenceServiceStub(channel)
    try:
        yield stub, backend
    finally:
        channel.close()
        server.stop(grace=None)


@pytest.fixture(scope="module")
def grpc_service() -> Iterator[inference_pb2_grpc.InferenceServiceStub]:
    """Grpc_service spins up an in-process gRPC test server with a stub backend."""
    backend = StubForecastBackend(
        model_name="test-stub",
        pipeline=object(),
        recommended_horizon=8,
    )
    with _start_servicer(backend) as (stub, _backend):
        yield stub


def test_grpc_get_model_info(grpc_service: inference_pb2_grpc.InferenceServiceStub) -> None:
    """Test_grpc_get_model_info verifies retrieval of model metadata."""
    response = grpc_service.GetModelInfo(inference_pb2.GetModelInfoRequest())

    assert response.model_name == "test-stub"
    assert response.max_horizon_hours == 240


def test_grpc_predict_hourly_weather_success(grpc_service: inference_pb2_grpc.InferenceServiceStub) -> None:
    """Test_grpc_predict_hourly_weather_success tests successful 24-hour forecasting over gRPC."""
    request = _build_hourly_request()
    response = grpc_service.PredictHourlyWeather(request)

    assert len(response.predictions) == 24
    assert response.HasField("computed_at")
    assert response.predictions[0].node_id == "station-alpha"
    assert response.predictions[0].horizon_hours == 1
    assert response.predictions[-1].horizon_hours == 24
    assert response.predictions[0].HasField("temperature_c")
    assert response.predictions[0].HasField("temperature_c_p10")
    assert response.predictions[0].HasField("temperature_c_p90")
    assert response.predictions[0].HasField("humidity_pct")
    assert response.predictions[0].HasField("rainfall_mm")
    assert response.predictions[0].HasField("rain_probability")
    assert response.predictions[0].HasField("pressure_hpa")
    # issue_time must be floored to top of the hour (:00)
    assert response.predictions[0].issue_time.seconds == 1699999200


def test_grpc_predict_hourly_weather_with_downtime_holes(
    grpc_service: inference_pb2_grpc.InferenceServiceStub,
) -> None:
    """Test_grpc_predict_hourly_weather_with_downtime_holes verifies forecasting succeeds across downtime gaps."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    # 32 hours total window. Gaps at steps 10..15 (6 hours missing) and 30..31 (2 hours trailing downtime)
    observations = []
    for step_index in range(32):
        if 10 <= step_index <= 15 or step_index >= 30:
            continue
        timestamp = Timestamp()
        timestamp.FromSeconds(1700000000 - ((32 - step_index) * 3600))
        obs = inference_pb2.TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=20.0 + (step_index * 0.1),
            humidity_pct=55.0,
            rainfall_mm=0.2,
        )
        observations.append(obs)

    request = inference_pb2.PredictHourlyWeatherRequest(
        node_id="station-with-outages",
        issue_time=issue_time,
        horizon_hours=24,
        history=observations,
    )

    response = grpc_service.PredictHourlyWeather(request)
    assert len(response.predictions) == 24
    assert response.predictions[0].node_id == "station-with-outages"
    assert response.predictions[0].issue_time.seconds == 1699999200
    assert response.predictions[0].HasField("temperature_c")
    assert response.predictions[0].HasField("rainfall_mm")


def test_grpc_predict_hourly_weather_validation_error(grpc_service: inference_pb2_grpc.InferenceServiceStub) -> None:
    """Test_grpc_predict_hourly_weather_validation_error tests rejection of malformed hourly request."""
    request = inference_pb2.PredictHourlyWeatherRequest(
        node_id="",
        horizon_hours=0,
    )

    with pytest.raises(grpc.RpcError) as exc_info:
        grpc_service.PredictHourlyWeather(request)

    assert cast(grpc.Call, exc_info.value).code() == grpc.StatusCode.INVALID_ARGUMENT


def test_grpc_predict_daily_weather_validation_error(grpc_service: inference_pb2_grpc.InferenceServiceStub) -> None:
    """Test_grpc_predict_daily_weather_validation_error tests rejection of malformed daily request."""
    request = inference_pb2.PredictDailyWeatherRequest(
        node_id="",
        horizon_days=0,
    )

    with pytest.raises(grpc.RpcError) as exc_info:
        grpc_service.PredictDailyWeather(request)

    assert cast(grpc.Call, exc_info.value).code() == grpc.StatusCode.INVALID_ARGUMENT


def test_grpc_predict_daily_weather_success(grpc_service: inference_pb2_grpc.InferenceServiceStub) -> None:
    """Test_grpc_predict_daily_weather_success tests successful 10-day forecasting over gRPC."""
    request = _build_daily_request()

    response = grpc_service.PredictDailyWeather(request)
    assert len(response.predictions) == 10
    assert response.predictions[0].horizon_days == 0
    assert response.predictions[-1].horizon_days == 9
    # issue_time floored to 00:00:00 UTC (1700000000 // 86400 * 86400 == 1699920000)
    assert response.predictions[0].issue_time.seconds == 1699920000
    assert response.predictions[0].HasField("pressure_hpa")


def test_grpc_predict_weather_warns_when_horizon_exceeds_recommendation(
    grpc_service: inference_pb2_grpc.InferenceServiceStub,
    caplog: pytest.LogCaptureFixture,
) -> None:
    """Test_grpc_predict_weather_warns_when_horizon_exceeds_recommendation logs an explicit horizon warning."""
    request = _build_hourly_request(include_optional=False)

    with caplog.at_level(logging.WARNING, logger="inference.server"):
        response = grpc_service.PredictHourlyWeather(request)

    assert len(response.predictions) == 24
    assert any("exceeds the recommended maximum" in record.message for record in caplog.records)


def test_grpc_predict_weather_no_horizon_warning_when_unbounded(
    caplog: pytest.LogCaptureFixture,
) -> None:
    """Test_grpc_predict_weather_no_horizon_warning_when_unbounded verifies no cap warning is logged."""
    backend = StubForecastBackend(model_name="test-unbounded", pipeline=object(), recommended_horizon=None)
    request = _build_hourly_request(include_optional=False)

    with _start_servicer(backend) as (stub, _backend), caplog.at_level(logging.WARNING, logger="inference.server"):
        response = stub.PredictHourlyWeather(request)

    assert len(response.predictions) == 24
    assert not any("exceeds the recommended maximum" in record.message for record in caplog.records)


def test_grpc_predict_hourly_weather_internal_error() -> None:
    """Test_grpc_predict_hourly_weather_internal_error maps backend failure to StatusCode.INTERNAL."""
    backend = StubForecastBackend(pipeline=object(), predict_error=RuntimeError("boom"))
    request = _build_hourly_request()

    with _start_servicer(backend) as (stub, _backend), pytest.raises(grpc.RpcError) as exc_info:
        stub.PredictHourlyWeather(request)

    assert cast(grpc.Call, exc_info.value).code() == grpc.StatusCode.INTERNAL
    assert "inference execution failed: boom" in (cast(grpc.Call, exc_info.value).details() or "")


def test_grpc_predict_daily_weather_internal_error() -> None:
    """Test_grpc_predict_daily_weather_internal_error maps daily backend failure to StatusCode.INTERNAL."""
    backend = StubForecastBackend(pipeline=object(), predict_error=RuntimeError("boom"))
    request = _build_daily_request()

    with _start_servicer(backend) as (stub, _backend), pytest.raises(grpc.RpcError) as exc_info:
        stub.PredictDailyWeather(request)

    assert cast(grpc.Call, exc_info.value).code() == grpc.StatusCode.INTERNAL
    assert "inference execution failed: boom" in (cast(grpc.Call, exc_info.value).details() or "")


@pytest.mark.parametrize(
    ("latitude", "longitude", "elevation_m"),
    [
        (45.5, None, None),
        (None, None, 500.0),
        (None, None, None),
    ],
)
def test_grpc_predict_hourly_partial_and_missing_location(
    latitude: float | None,
    longitude: float | None,
    elevation_m: float | None,
) -> None:
    """Test_grpc_predict_hourly_partial_and_missing_location handles omitted optional location fields."""
    backend = StubForecastBackend(pipeline=object())
    request = _build_hourly_request(latitude=latitude, longitude=longitude, elevation_m=elevation_m)
    zero_lon_request = _build_hourly_request(latitude=latitude, longitude=0.0, elevation_m=elevation_m)
    shifted_lon_request = _build_hourly_request(latitude=latitude, longitude=9.2, elevation_m=elevation_m)

    with _start_servicer(backend) as (stub, active_backend):
        response = stub.PredictHourlyWeather(request)
        default_future = cast(dict[str, np.ndarray], active_backend.captured_calls[-1]["future_covariates"])
        stub.PredictHourlyWeather(zero_lon_request)
        zero_lon_future = cast(dict[str, np.ndarray], active_backend.captured_calls[-1]["future_covariates"])
        stub.PredictHourlyWeather(shifted_lon_request)
        shifted_lon_future = cast(dict[str, np.ndarray], active_backend.captured_calls[-1]["future_covariates"])

    assert len(response.predictions) == 24
    assert all(np.isfinite(pred.temperature_c) for pred in response.predictions)
    np.testing.assert_allclose(default_future["solar_hour_sin"], zero_lon_future["solar_hour_sin"])
    assert not np.allclose(default_future["solar_hour_sin"], shifted_lon_future["solar_hour_sin"])


@pytest.mark.parametrize(
    ("latitude", "longitude", "elevation_m"),
    [
        (45.5, None, None),
        (None, None, None),
    ],
)
def test_grpc_predict_daily_partial_and_missing_location(
    latitude: float | None,
    longitude: float | None,
    elevation_m: float | None,
) -> None:
    """Test_grpc_predict_daily_partial_and_missing_location handles omitted daily location fields."""
    backend = StubForecastBackend(pipeline=object())
    request = _build_daily_request(latitude=latitude, longitude=longitude, elevation_m=elevation_m)

    with _start_servicer(backend) as (stub, active_backend):
        response = stub.PredictDailyWeather(request)
        captured_future = cast(dict[str, np.ndarray], active_backend.captured_calls[-1]["future_covariates"])

    assert len(response.predictions) == 10
    assert all(np.isfinite(pred.min_temperature_c) for pred in response.predictions)
    for key in ("clear_sky_radiation_mj", "daylight_hours", "doy_sin", "doy_cos"):
        assert key in captured_future
        assert np.isfinite(captured_future[key]).all()


def test_server_lifecycle() -> None:
    """Test_server_lifecycle verifies start, health checks, and stop of InferenceServer."""
    config = Config(host="127.0.0.1", port=0, model_name="test-stub")
    backend = StubForecastBackend(model_name="test-stub", pipeline=object())

    server = InferenceServer(config=config, backend=backend)
    server.start()

    channel = grpc.insecure_channel(
        f"127.0.0.1:{server.bound_port}",
        options=[("grpc.enable_http_proxy", 0)],
    )
    health_stub = health_pb2_grpc.HealthStub(channel)

    health_resp = health_stub.Check(health_pb2.HealthCheckRequest(service=""))
    assert health_resp.status == health_pb2.HealthCheckResponse.SERVING

    ready_resp = health_stub.Check(health_pb2.HealthCheckRequest(service="ready"))
    assert ready_resp.status == health_pb2.HealthCheckResponse.SERVING

    channel.close()
    server.stop(grace=0.1)


def test_server_idle_eviction_loop(caplog: pytest.LogCaptureFixture) -> None:
    """Test_server_idle_eviction_loop evicts an idle preloaded pipeline on schedule."""
    unloaded_event = threading.Event()
    backend = StubForecastBackend(
        model_name="test-idle",
        idle_timeout_seconds=0.0,
        pipeline=object(),
        on_unload=unloaded_event,
    )
    config = Config(host="127.0.0.1", port=0, model_name="test-idle")
    server = InferenceServer(config=config, backend=backend, idle_check_interval_seconds=0.01)

    with caplog.at_level(logging.INFO, logger="inference.server"):
        server.start()
        try:
            assert unloaded_event.wait(timeout=2.0)
            assert not backend.is_loaded()
        finally:
            server.stop(grace=0.1)
            server._idle_thread.join(timeout=1.0)

    assert any("idle model successfully evicted from memory" in record.message for record in caplog.records)
