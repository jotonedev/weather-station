from concurrent import futures

import grpc
import pytest
from google.protobuf.timestamp_pb2 import Timestamp
from grpc_health.v1 import health_pb2, health_pb2_grpc
from weather.inference.v1 import inference_pb2, inference_pb2_grpc

from inference.backends.chronos import Chronos2Backend
from inference.config import Config
from inference.server import InferenceServer, InferenceServicer
from tests.helpers import create_synthetic_chronos_pipeline


@pytest.fixture(scope="module")
def grpc_service():
    """Grpc_service spins up an in-process gRPC test server with synthetic pipeline."""
    synthetic_pipeline = create_synthetic_chronos_pipeline()
    config = Config(host="127.0.0.1", port=0, model_name="test-chronos", max_horizon_hours=240)
    backend = Chronos2Backend(
        model_name="test-chronos",
        pipeline=synthetic_pipeline,
    )

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    servicer = InferenceServicer(backend, config)
    inference_pb2_grpc.add_InferenceServiceServicer_to_server(servicer, server)

    port = server.add_insecure_port("127.0.0.1:0")
    server.start()

    channel = grpc.insecure_channel(f"127.0.0.1:{port}")
    stub = inference_pb2_grpc.InferenceServiceStub(channel)

    yield stub

    channel.close()
    server.stop(grace=None)


def test_grpc_get_model_info(grpc_service: inference_pb2_grpc.InferenceServiceStub) -> None:
    """Test_grpc_get_model_info verifies retrieval of model metadata."""
    request = inference_pb2.GetModelInfoRequest()
    response = grpc_service.GetModelInfo(request)

    assert response.model_name == "test-chronos"
    assert response.max_horizon_hours == 240


def test_grpc_predict_weather_success(grpc_service: inference_pb2_grpc.InferenceServiceStub) -> None:
    """Test_grpc_predict_weather_success tests successful 24-hour forecasting over gRPC."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    # Request mirrors the production shape produced by the Go inference-job
    # mapper: temperature, humidity and pressure only (no wind, no rainfall).
    observations = []
    for step_index in range(32):
        timestamp = Timestamp()
        timestamp.FromSeconds(1700000000 - ((32 - step_index) * 3600))
        obs = inference_pb2.TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=22.0 + (step_index * 0.1),
            humidity_pct=60.0,
            pressure_hpa=1013.0,
        )
        observations.append(obs)

    request = inference_pb2.PredictWeatherRequest(
        node_id="station-alpha",
        mode=inference_pb2.PREDICTION_MODE_NEXT_24_HOURS,
        issue_time=issue_time,
        history=observations,
    )

    response = grpc_service.PredictWeather(request)

    assert len(response.predictions) == 24
    assert response.HasField("computed_at")
    assert response.predictions[0].node_id == "station-alpha"
    assert response.predictions[0].horizon_hours == 1
    assert response.predictions[-1].horizon_hours == 24
    assert not response.predictions[0].HasField("wind_direction_deg")
    assert response.predictions[0].HasField("temperature_c")
    assert response.predictions[0].HasField("humidity_pct")


def test_grpc_predict_weather_validation_error(grpc_service: inference_pb2_grpc.InferenceServiceStub) -> None:
    """Test_grpc_predict_weather_validation_error tests rejection of malformed request."""
    request = inference_pb2.PredictWeatherRequest(
        node_id="",
        mode=inference_pb2.PREDICTION_MODE_UNSPECIFIED,
    )

    with pytest.raises(grpc.RpcError) as exc_info:
        grpc_service.PredictWeather(request)

    assert isinstance(exc_info.value, grpc.Call)
    assert exc_info.value.code() == grpc.StatusCode.INVALID_ARGUMENT


def test_grpc_predict_weather_daily_mode(grpc_service: inference_pb2_grpc.InferenceServiceStub) -> None:
    """Test_grpc_predict_weather_daily_mode tests successful 10-day forecasting over gRPC."""
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
        )
        observations.append(obs)

    request = inference_pb2.PredictWeatherRequest(
        node_id="station-beta",
        mode=inference_pb2.PREDICTION_MODE_NEXT_10_DAYS,
        issue_time=issue_time,
        history=observations,
    )

    response = grpc_service.PredictWeather(request)
    assert len(response.predictions) == 10
    assert response.predictions[0].horizon_hours == 24
    assert response.predictions[-1].horizon_hours == 240


def test_config_from_env(monkeypatch: pytest.MonkeyPatch) -> None:
    """Test_config_from_env verifies parsing of environment variables."""
    monkeypatch.setenv("INFERENCE_PORT", "60060")
    monkeypatch.setenv("INFERENCE_HOST", "127.0.0.1")
    monkeypatch.setenv("INFERENCE_MODEL_BACKEND", "chronos-2")
    monkeypatch.setenv("INFERENCE_MODEL_NAME", "custom-model")
    monkeypatch.setenv("INFERENCE_IDLE_TIMEOUT_SECONDS", "1800")
    monkeypatch.setenv("INFERENCE_MAX_WORKERS", "8")
    monkeypatch.setenv("INFERENCE_MAX_HORIZON_HOURS", "360")

    config = Config.from_env()
    assert config.port == 60060
    assert config.host == "127.0.0.1"
    assert config.model_backend == "chronos-2"
    assert config.model_name == "custom-model"
    assert config.idle_timeout_seconds == 1800.0
    assert config.max_workers == 8
    assert config.max_horizon_hours == 360


def test_server_lifecycle() -> None:
    """Test_server_lifecycle verifies start, health checks, and stop of InferenceServer."""
    synthetic_pipeline = create_synthetic_chronos_pipeline()
    config = Config(host="127.0.0.1", port=0, model_name="test-chronos")
    backend = Chronos2Backend(
        model_name="test-chronos",
        pipeline=synthetic_pipeline,
    )

    server = InferenceServer(config=config, backend=backend)
    server.start()

    channel = grpc.insecure_channel(f"127.0.0.1:{server.bound_port}")
    health_stub = health_pb2_grpc.HealthStub(channel)

    health_resp = health_stub.Check(health_pb2.HealthCheckRequest(service=""))
    assert health_resp.status == health_pb2.HealthCheckResponse.SERVING

    ready_resp = health_stub.Check(health_pb2.HealthCheckRequest(service="ready"))
    assert ready_resp.status == health_pb2.HealthCheckResponse.SERVING

    channel.close()
    server.stop(grace=0.1)
