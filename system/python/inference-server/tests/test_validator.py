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


@pytest.mark.parametrize("issue_seconds", [None, 0, -100])
def test_validate_request_missing_issue_time(issue_seconds: int | None) -> None:
    """Test_validate_request_missing_issue_time checks failure when issue time is unset or non-positive."""
    request = create_valid_hourly_request()
    if issue_seconds is None:
        request.issue_time.Clear()
    else:
        request.issue_time.FromSeconds(issue_seconds)
    with pytest.raises(ValidationError, match="valid issue_time is required"):
        validate_hourly_request(request)


def test_validate_request_empty_history() -> None:
    """Test_validate_request_empty_history checks failure on empty observation list."""
    request = create_valid_hourly_request()
    del request.history[:]
    with pytest.raises(ValidationError, match="history observations cannot be empty"):
        validate_hourly_request(request)


@pytest.mark.parametrize("field_name", ["temperature_c", "humidity_pct"])
def test_validate_request_required_field_sparsity(field_name: str) -> None:
    """Test_validate_request_required_field_sparsity enforces the 50% missing threshold."""
    request = create_valid_hourly_request()
    for index in range(5):
        request.history[index].ClearField(field_name)
    validate_hourly_request(request)

    request.history[5].ClearField(field_name)
    with pytest.raises(ValidationError, match=f"{field_name} exceeds maximum allowed missing ratio"):
        validate_hourly_request(request)


def test_validate_request_trailing_downtime() -> None:
    """Test_validate_request_trailing_downtime checks missing ratio on trailing station outage."""
    request = create_valid_hourly_request()
    del request.history[4:]
    with pytest.raises(ValidationError, match="temperature_c exceeds maximum allowed missing ratio"):
        validate_hourly_request(request)

    request_pass = create_valid_hourly_request()
    del request_pass.history[6:]
    validate_hourly_request(request_pass)


def test_validate_request_early_outage_trimmed() -> None:
    """Test_validate_request_early_outage_trimmed ensures old outage is trimmed when recent suffix is healthy."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    observations = []
    for step in range(2):
        ts = Timestamp()
        ts.FromSeconds(1700000000 - (20 * 86400) + (step * 3600))
        observations.append(TimeSeriesObservation(timestamp=ts, temperature_c=18.0, humidity_pct=50.0))

    for step in range(10):
        ts = Timestamp()
        ts.FromSeconds(1700000000 - ((10 - step) * 3600))
        observations.append(TimeSeriesObservation(timestamp=ts, temperature_c=20.0, humidity_pct=55.0))

    request = PredictHourlyWeatherRequest(
        node_id="station-node-alpha",
        issue_time=issue_time,
        horizon_hours=24,
        history=observations,
    )
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
        ("wind_direction_deg", -0.1),
        ("wind_direction_deg", 360.1),
        ("max_wind_speed_mps", -1.0),
        ("max_wind_speed_mps", 160.0),
        ("wind_u_mps", -160.0),
        ("wind_u_mps", 160.0),
        ("wind_v_mps", -160.0),
        ("wind_v_mps", 160.0),
        ("max_rain_rate_mmh", -0.5),
        ("max_rain_rate_mmh", 600.0),
        ("rainfall_mm", -0.5),
    ],
)
def test_validate_request_physical_bounds(field_name: str, invalid_value: float) -> None:
    """Test_validate_request_physical_bounds checks rejection of values outside meteorological limits."""
    request = create_valid_hourly_request()
    setattr(request.history[0], field_name, invalid_value)

    with pytest.raises(ValidationError, match=r"out of bounds"):
        validate_hourly_request(request)


@pytest.mark.parametrize(
    ("field_name", "invalid_value"),
    [
        ("min_temperature_c", -95.0),
        ("min_temperature_c", 75.0),
        ("max_temperature_c", -95.0),
        ("max_temperature_c", 75.0),
        ("min_humidity_pct", -5.0),
        ("min_humidity_pct", 105.0),
        ("max_humidity_pct", -5.0),
        ("max_humidity_pct", 105.0),
        ("min_pressure_hpa", 250.0),
        ("min_pressure_hpa", 1150.0),
        ("max_pressure_hpa", 250.0),
        ("max_pressure_hpa", 1150.0),
    ],
)
def test_validate_daily_request_physical_bounds(field_name: str, invalid_value: float) -> None:
    """Test_validate_daily_request_physical_bounds rejects out-of-range daily min/max sensor fields."""
    request = create_valid_daily_request()
    setattr(request.history[0], field_name, invalid_value)

    with pytest.raises(ValidationError, match=r"out of bounds"):
        validate_daily_request(request)


def test_validate_request_rejects_zero_completed_observations() -> None:
    """Test_validate_request_rejects_zero_completed_observations rejects history only at/after issue_time."""
    request = create_valid_hourly_request()
    issue_sec = request.issue_time.seconds
    del request.history[:]
    ts = Timestamp()
    ts.FromSeconds(issue_sec)
    request.history.append(TimeSeriesObservation(timestamp=ts, temperature_c=20.0, humidity_pct=50.0))

    with pytest.raises(ValidationError, match="no completed observations"):
        validate_hourly_request(request)


def test_validate_request_deduplicates_same_bucket_timestamps() -> None:
    """Test_validate_request_deduplicates_same_bucket_timestamps prevents duplicate readings masking sparsity."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)
    floored_issue = (issue_time.seconds // 3600) * 3600

    observations = []
    # 10 duplicate observations in the earliest bucket (10 hours before issue_time) and nothing in the next 9 buckets
    for offset in range(10):
        ts = Timestamp()
        ts.FromSeconds(floored_issue - (10 * 3600) + (offset * 60))
        observations.append(TimeSeriesObservation(timestamp=ts, temperature_c=20.0, humidity_pct=50.0))

    request = PredictHourlyWeatherRequest(
        node_id="station-node-alpha",
        issue_time=issue_time,
        horizon_hours=24,
        history=observations,
    )
    with pytest.raises(ValidationError, match="exceeds maximum allowed missing ratio"):
        validate_hourly_request(request)


@pytest.mark.parametrize(
    "field_name",
    ["min_temperature_c", "max_temperature_c", "min_humidity_pct", "max_humidity_pct"],
)
def test_validate_daily_request_checks_min_max_sparsity(field_name: str) -> None:
    """Test_validate_daily_request_checks_min_max_sparsity rejects >50% missing daily min/max targets."""
    request = create_valid_daily_request()
    for index in range(6):
        request.history[index].ClearField(field_name)

    with pytest.raises(ValidationError, match=f"{field_name} exceeds maximum allowed missing ratio"):
        validate_daily_request(request)


def test_validate_request_non_finite_treated_as_missing() -> None:
    """Test_validate_request_non_finite_treated_as_missing skips bounds check and counts toward sparsity."""
    request = create_valid_hourly_request()
    request.history[0].temperature_c = float("nan")
    request.history[0].pressure_hpa = float("inf")
    request.history[1].humidity_pct = float("-inf")
    validate_hourly_request(request)

    for non_finite_val in (float("nan"), float("inf")):
        sparse_request = create_valid_hourly_request()
        for index in range(6):
            sparse_request.history[index].temperature_c = non_finite_val
        with pytest.raises(ValidationError, match="temperature_c exceeds maximum allowed missing ratio"):
            validate_hourly_request(sparse_request)


def test_validate_request_bucket_deduplication_keeps_latest() -> None:
    """Test_validate_request_bucket_deduplication_keeps_latest uses chronological latest per bucket."""
    issue_time = Timestamp(seconds=1700000000)
    floored_issue = (issue_time.seconds // 3600) * 3600

    failing_observations = []
    passing_observations = []
    for bucket_offset in range(10):
        bucket_start = floored_issue - ((10 - bucket_offset) * 3600)
        early_ts = Timestamp(seconds=bucket_start)
        late_ts = Timestamp(seconds=bucket_start + 300)

        if bucket_offset < 6:
            failing_observations.append(TimeSeriesObservation(timestamp=late_ts, humidity_pct=50.0))
            failing_observations.append(
                TimeSeriesObservation(timestamp=early_ts, temperature_c=20.0, humidity_pct=50.0)
            )
            passing_observations.append(TimeSeriesObservation(timestamp=early_ts, humidity_pct=50.0))
            passing_observations.append(
                TimeSeriesObservation(timestamp=late_ts, temperature_c=20.0, humidity_pct=50.0)
            )
        else:
            failing_observations.append(
                TimeSeriesObservation(timestamp=early_ts, temperature_c=20.0, humidity_pct=50.0)
            )
            passing_observations.append(
                TimeSeriesObservation(timestamp=early_ts, temperature_c=20.0, humidity_pct=50.0)
            )

    failing_request = PredictHourlyWeatherRequest(
        node_id="station-node-alpha",
        issue_time=issue_time,
        horizon_hours=24,
        history=failing_observations,
    )
    with pytest.raises(ValidationError, match="temperature_c exceeds maximum allowed missing ratio"):
        validate_hourly_request(failing_request)

    passing_request = PredictHourlyWeatherRequest(
        node_id="station-node-alpha",
        issue_time=issue_time,
        horizon_hours=24,
        history=passing_observations,
    )
    validate_hourly_request(passing_request)


def test_validate_request_early_outage_multi_gap_and_unrecoverable() -> None:
    """Test_validate_request_early_outage_multi_gap_and_unrecoverable scans gaps until healthy or fails."""
    issue_time = Timestamp(seconds=1700000000)

    # Gap 1 -> sparse middle cluster (still >50% missing over suffix); Gap 2 -> healthy recent suffix
    observations = [
        TimeSeriesObservation(
            timestamp=Timestamp(seconds=1700000000 - (20 * 86400)),
            temperature_c=18.0,
            humidity_pct=50.0,
        ),
        TimeSeriesObservation(
            timestamp=Timestamp(seconds=1700000000 - (10 * 86400)),
            temperature_c=19.0,
            humidity_pct=50.0,
        ),
    ]
    for step in range(10):
        observations.append(
            TimeSeriesObservation(
                timestamp=Timestamp(seconds=1700000000 - ((10 - step) * 3600)),
                temperature_c=20.0,
                humidity_pct=55.0,
            )
        )

    validate_hourly_request(
        PredictHourlyWeatherRequest(
            node_id="station-node-alpha",
            issue_time=issue_time,
            horizon_hours=24,
            history=observations,
        )
    )

    unrecoverable_observations = [
        TimeSeriesObservation(
            timestamp=Timestamp(seconds=1700000000 - (20 * 86400)),
            temperature_c=18.0,
            humidity_pct=50.0,
        ),
        TimeSeriesObservation(
            timestamp=Timestamp(seconds=1700000000 - (10 * 3600)),
            temperature_c=19.0,
            humidity_pct=50.0,
        ),
    ]
    with pytest.raises(ValidationError, match="temperature_c exceeds maximum allowed missing ratio"):
        validate_hourly_request(
            PredictHourlyWeatherRequest(
                node_id="station-node-alpha",
                issue_time=issue_time,
                horizon_hours=24,
                history=unrecoverable_observations,
            )
        )
