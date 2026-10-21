import math

from weather.inference.v1.inference_pb2 import (
    PredictDailyWeatherRequest,
    PredictHourlyWeatherRequest,
    TimeSeriesObservation,
)


class ValidationError(ValueError):
    """ValidationError indicates malformed or out of bounds request data."""


# Only fields actually used by the model are validated. Fields that are
# received but not modeled (e.g. wind speed) must not reject a forecast.
SENSOR_BOUNDS = (
    ("temperature_c", -90.0, 70.0),
    ("min_temperature_c", -90.0, 70.0),
    ("max_temperature_c", -90.0, 70.0),
    ("humidity_pct", 0.0, 100.0),
    ("min_humidity_pct", 0.0, 100.0),
    ("max_humidity_pct", 0.0, 100.0),
    ("pressure_hpa", 300.0, 1100.0),
    ("min_pressure_hpa", 300.0, 1100.0),
    ("max_pressure_hpa", 300.0, 1100.0),
    ("rainfall_mm", 0.0, float("inf")),
    ("wind_speed_mps", 0.0, 150.0),
    ("wind_direction_deg", 0.0, 360.0),
)


def validate_observation_bounds(observation: TimeSeriesObservation, index: int) -> None:
    """Validate_observation_bounds checks physical sensor boundaries for finite values."""
    for field_name, lower_bound, upper_bound in SENSOR_BOUNDS:
        if not observation.HasField(field_name):
            continue
        value = getattr(observation, field_name)
        # Non-finite values (NaN or +/-inf) are treated as missing data, not as
        # measurements: they are counted by the sparsity checks and imputed or
        # rejected there, rather than failing the whole request. Only finite
        # values are range-checked.
        if not math.isfinite(value):
            continue
        if value < lower_bound or value > upper_bound:
            raise ValidationError(f"{field_name} at index {index} out of bounds: {value}")


def _validate_common_request(request: PredictHourlyWeatherRequest | PredictDailyWeatherRequest) -> None:
    if not request.node_id or not request.node_id.strip():
        raise ValidationError("node_id is required")

    if not request.HasField("issue_time") or request.issue_time.seconds <= 0:
        raise ValidationError("valid issue_time is required")

    total_observations = len(request.history)
    if total_observations == 0:
        raise ValidationError("history observations cannot be empty")

    def is_effectively_missing(obs: TimeSeriesObservation, field_name: str) -> bool:
        """is_effectively_missing treats absent and non-finite sensor values as missing."""
        return not obs.HasField(field_name) or not math.isfinite(getattr(obs, field_name))

    missing_temp_count = sum(1 for obs in request.history if is_effectively_missing(obs, "temperature_c"))
    missing_hum_count = sum(1 for obs in request.history if is_effectively_missing(obs, "humidity_pct"))

    if (missing_temp_count / total_observations) > 0.30:
        raise ValidationError("temperature_c exceeds maximum allowed missing ratio of 30 percent")

    if (missing_hum_count / total_observations) > 0.30:
        raise ValidationError("humidity_pct exceeds maximum allowed missing ratio of 30 percent")

    for obs_index, obs in enumerate(request.history):
        validate_observation_bounds(obs, obs_index)


def validate_hourly_request(request: PredictHourlyWeatherRequest) -> None:
    """validate_hourly_request verifies incoming request headers, horizon, and historical observations."""
    if request.horizon_hours <= 0:
        raise ValidationError(f"horizon_hours must be positive: {request.horizon_hours}")
    if request.horizon_hours > 168:
        raise ValidationError(f"horizon_hours exceeds maximum allowed of 168: {request.horizon_hours}")
    _validate_common_request(request)


def validate_daily_request(request: PredictDailyWeatherRequest) -> None:
    """validate_daily_request verifies incoming request headers, horizon, and historical observations."""
    if request.horizon_days <= 0:
        raise ValidationError(f"horizon_days must be positive: {request.horizon_days}")
    if request.horizon_days > 30:
        raise ValidationError(f"horizon_days exceeds maximum allowed of 30: {request.horizon_days}")
    _validate_common_request(request)
