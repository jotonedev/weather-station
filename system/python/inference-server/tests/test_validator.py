import pytest
from google.protobuf.timestamp_pb2 import Timestamp
from weather.inference.v1.inference_pb2 import (
    PREDICTION_MODE_NEXT_24_HOURS,
    PREDICTION_MODE_UNSPECIFIED,
    PredictWeatherRequest,
    TimeSeriesObservation,
)

from inference.validator import ValidationError, validate_request


def create_valid_request() -> PredictWeatherRequest:
    """Create_valid_request builds a valid request for testing."""
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

    return PredictWeatherRequest(
        node_id="station-node-alpha",
        mode=PREDICTION_MODE_NEXT_24_HOURS,
        issue_time=issue_time,
        history=observations,
    )


def test_validate_request_success() -> None:
    """Test_validate_request_success verifies that a well formed request passes."""
    request = create_valid_request()
    validate_request(request)


def test_validate_request_missing_node_id() -> None:
    """Test_validate_request_missing_node_id checks failure on blank node id."""
    request = create_valid_request()
    request.node_id = "   "
    with pytest.raises(ValidationError, match="node_id is required"):
        validate_request(request)


def test_validate_request_invalid_mode() -> None:
    """Test_validate_request_invalid_mode checks failure on unsupported mode."""
    request = create_valid_request()
    request.mode = PREDICTION_MODE_UNSPECIFIED
    with pytest.raises(ValidationError, match="unsupported prediction mode"):
        validate_request(request)


def test_validate_request_missing_issue_time() -> None:
    """Test_validate_request_missing_issue_time checks failure when issue time is zero."""
    request = create_valid_request()
    request.issue_time.Clear()
    with pytest.raises(ValidationError, match="valid issue_time is required"):
        validate_request(request)


def test_validate_request_empty_history() -> None:
    """Test_validate_request_empty_history checks failure on empty observation list."""
    request = create_valid_request()
    del request.history[:]
    with pytest.raises(ValidationError, match="history observations cannot be empty"):
        validate_request(request)


def test_validate_request_temperature_sparsity() -> None:
    """Test_validate_request_temperature_sparsity checks failure when temperature missing exceeds 30 percent."""
    request = create_valid_request()
    for index in range(4):
        request.history[index].ClearField("temperature_c")

    with pytest.raises(ValidationError, match="temperature_c exceeds maximum allowed missing ratio"):
        validate_request(request)


def test_validate_request_humidity_sparsity() -> None:
    """Test_validate_request_humidity_sparsity checks failure when humidity missing exceeds 30 percent."""
    request = create_valid_request()
    for index in range(4):
        request.history[index].ClearField("humidity_pct")

    with pytest.raises(ValidationError, match="humidity_pct exceeds maximum allowed missing ratio"):
        validate_request(request)


@pytest.mark.parametrize(
    ("field_name", "invalid_value"),
    [
        ("temperature_c", -100.0),
        ("temperature_c", 80.0),
        ("humidity_pct", -5.0),
        ("humidity_pct", 105.0),
        ("pressure_hpa", 250.0),
        ("pressure_hpa", 1200.0),
        ("rainfall_mm", -0.5),
    ],
)
def test_validate_request_out_of_bounds(field_name: str, invalid_value: float) -> None:
    """Test_validate_request_out_of_bounds verifies boundaries for sensor measurements."""
    request = create_valid_request()
    setattr(request.history[0], field_name, invalid_value)
    with pytest.raises(ValidationError, match="out of bounds"):
        validate_request(request)


@pytest.mark.parametrize(
    ("field_name", "invalid_value"),
    [
        ("temperature_c", float("inf")),
        ("temperature_c", float("-inf")),
        ("humidity_pct", float("inf")),
    ],
)
def test_validate_request_inf_over_threshold_rejected(field_name: str, invalid_value: float) -> None:
    """Test_validate_request_inf_over_threshold_rejected rejects required columns dominated by infinities."""
    request = create_valid_request()
    # 4 of 10 non-finite (40 percent) exceeds the 30 percent missing threshold.
    for index in range(4):
        setattr(request.history[index], field_name, invalid_value)

    with pytest.raises(ValidationError, match="exceeds maximum allowed missing ratio"):
        validate_request(request)


def test_validate_request_all_nan_temperature_rejected() -> None:
    """Test_validate_request_all_nan_temperature_rejected treats NaN as missing for the sparsity check."""
    request = create_valid_request()
    for observation in request.history:
        observation.temperature_c = float("nan")

    with pytest.raises(ValidationError, match="temperature_c exceeds maximum allowed missing ratio"):
        validate_request(request)


def test_validate_request_all_nan_humidity_rejected() -> None:
    """Test_validate_request_all_nan_humidity_rejected treats NaN as missing for the sparsity check."""
    request = create_valid_request()
    for observation in request.history:
        observation.humidity_pct = float("nan")

    with pytest.raises(ValidationError, match="humidity_pct exceeds maximum allowed missing ratio"):
        validate_request(request)


def test_validate_request_partial_nan_accepted() -> None:
    """Test_validate_request_partial_nan_accepted allows non-finite values within the missing threshold."""
    request = create_valid_request()
    # 2 of 10 missing (20 percent) is under the 30 percent threshold.
    request.history[0].temperature_c = float("nan")
    request.history[1].humidity_pct = float("nan")

    validate_request(request)


def test_validate_request_present_and_nan_counts_once() -> None:
    """Test_validate_request_present_and_nan_counts_once verifies a NaN field is not double counted."""
    request = create_valid_request()
    # 4 NaN plus 0 cleared means 4 of 10 (40 percent) - over threshold, rejected.
    for index in range(4):
        request.history[index].temperature_c = float("nan")

    with pytest.raises(ValidationError, match="temperature_c exceeds maximum allowed missing ratio"):
        validate_request(request)
