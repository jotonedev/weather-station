import pytest
from google.protobuf.timestamp_pb2 import Timestamp
from weather.inference.v1.inference_pb2 import (
    PredictDailyWeatherRequest,
    PredictHourlyWeatherRequest,
    TimeSeriesObservation,
)

from inference.validator import (
    ValidationError,
    validate_daily_request,
    validate_hourly_request,
)


def create_valid_hourly_request() -> PredictHourlyWeatherRequest:
    """Create_valid_hourly_request builds a valid hourly request for testing."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    observations = []
    for step_index in range(10):
        timestamp = Timestamp()
        timestamp.FromSeconds(1700000000 - ((10 - step_index) * 3600))
        observation = TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=21.5,
            humidity_pct=55.0,
            pressure_hpa=1013.2,
            wind_speed_mps=3.5,
            wind_direction_deg=180.0,
            rainfall_mm=0.0,
        )
        observations.append(observation)

    return PredictHourlyWeatherRequest(
        node_id="station-node-alpha",
        issue_time=issue_time,
        horizon_hours=24,
        history=observations,
    )


def create_valid_daily_request() -> PredictDailyWeatherRequest:
    """Create_valid_daily_request builds a valid daily request for testing."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    observations = []
    for step_index in range(10):
        timestamp = Timestamp()
        timestamp.FromSeconds(1700000000 - ((10 - step_index) * 86400))
        observation = TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=21.5,
            humidity_pct=55.0,
            min_temperature_c=15.0,
            max_temperature_c=25.0,
            min_humidity_pct=40.0,
            max_humidity_pct=70.0,
            rainfall_mm=0.0,
        )
        observations.append(observation)

    return PredictDailyWeatherRequest(
        node_id="station-node-beta",
        issue_time=issue_time,
        horizon_days=10,
        history=observations,
    )


def test_validate_hourly_request_success() -> None:
    """Test_validate_hourly_request_success verifies that a well formed request passes."""
    request = create_valid_hourly_request()
    validate_hourly_request(request)


def test_validate_daily_request_success() -> None:
    """Test_validate_daily_request_success verifies that a well formed daily request passes."""
    request = create_valid_daily_request()
    validate_daily_request(request)


def test_validate_hourly_request_missing_node_id() -> None:
    """Test_validate_hourly_request_missing_node_id checks failure on blank node id."""
    request = create_valid_hourly_request()
    request.node_id = "   "
    with pytest.raises(ValidationError, match="node_id is required"):
        validate_hourly_request(request)


def test_validate_hourly_request_invalid_horizon() -> None:
    """Test_validate_hourly_request_invalid_horizon checks horizon_hours bounds."""
    request = create_valid_hourly_request()
    request.horizon_hours = 0
    with pytest.raises(ValidationError, match="horizon_hours must be positive"):
        validate_hourly_request(request)

    request.horizon_hours = 200
    with pytest.raises(ValidationError, match="horizon_hours exceeds maximum allowed of 168"):
        validate_hourly_request(request)


def test_validate_daily_request_invalid_horizon() -> None:
    """Test_validate_daily_request_invalid_horizon checks horizon_days bounds."""
    request = create_valid_daily_request()
    request.horizon_days = 0
    with pytest.raises(ValidationError, match="horizon_days must be positive"):
        validate_daily_request(request)

    request.horizon_days = 40
    with pytest.raises(ValidationError, match="horizon_days exceeds maximum allowed of 30"):
        validate_daily_request(request)


def test_validate_request_missing_issue_time() -> None:
    """Test_validate_request_missing_issue_time checks failure when issue time is zero."""
    request = create_valid_hourly_request()
    request.issue_time.Clear()
    with pytest.raises(ValidationError, match="valid issue_time is required"):
        validate_hourly_request(request)


def test_validate_request_empty_history() -> None:
    """Test_validate_request_empty_history checks failure on empty observation list."""
    request = create_valid_hourly_request()
    del request.history[:]
    with pytest.raises(ValidationError, match="history observations cannot be empty"):
        validate_hourly_request(request)


def test_validate_request_temperature_sparsity() -> None:
    """Test_validate_request_temperature_sparsity checks failure when temperature missing exceeds 30 percent."""
    request = create_valid_hourly_request()
    for index in range(4):
        request.history[index].ClearField("temperature_c")

    with pytest.raises(ValidationError, match="temperature_c exceeds maximum allowed missing ratio"):
        validate_hourly_request(request)


def test_validate_request_humidity_sparsity() -> None:
    """Test_validate_request_humidity_sparsity checks failure when humidity missing exceeds 30 percent."""
    request = create_valid_hourly_request()
    for index in range(4):
        request.history[index].ClearField("humidity_pct")

    with pytest.raises(ValidationError, match="humidity_pct exceeds maximum allowed missing ratio"):
        validate_hourly_request(request)


@pytest.mark.parametrize(
    ("field_name", "invalid_value"),
    [
        ("temperature_c", -95.0),
        ("temperature_c", 75.0),
        ("humidity_pct", -5.0),
        ("humidity_pct", 105.0),
        ("pressure_hpa", 250.0),
        ("pressure_hpa", 1150.0),
        ("wind_speed_mps", -1.0),
        ("wind_speed_mps", 160.0),
        ("rainfall_mm", -0.5),
    ],
)
def test_validate_request_physical_bounds(field_name: str, invalid_value: float) -> None:
    """Test_validate_request_physical_bounds checks rejection of values outside meteorological limits."""
    request = create_valid_hourly_request()
    setattr(request.history[0], field_name, invalid_value)

    with pytest.raises(ValidationError, match=r"out of bounds"):
        validate_hourly_request(request)
