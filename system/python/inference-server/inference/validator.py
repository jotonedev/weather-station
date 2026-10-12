from weather.inference.v1.inference_pb2 import (
    PREDICTION_MODE_NEXT_10_DAYS,
    PREDICTION_MODE_NEXT_24_HOURS,
    PredictWeatherRequest,
    TimeSeriesObservation,
)


class ValidationError(ValueError):
    """ValidationError indicates malformed or out of bounds request data."""


SENSOR_BOUNDS = (
    ("temperature_c", -90.0, 70.0),
    ("humidity_pct", 0.0, 100.0),
    ("pressure_hpa", 300.0, 1100.0),
    ("wind_speed_mps", 0.0, float("inf")),
    ("wind_direction_deg", 0.0, 360.0),
    ("rainfall_mm", 0.0, float("inf")),
)


def validate_observation_bounds(observation: TimeSeriesObservation, index: int) -> None:
    """Validate_observation_bounds checks physical sensor boundaries."""
    for field_name, lower_bound, upper_bound in SENSOR_BOUNDS:
        if not observation.HasField(field_name):
            continue
        value = getattr(observation, field_name)
        if value < lower_bound or value > upper_bound:
            raise ValidationError(f"{field_name} at index {index} out of bounds: {value}")


def validate_request(request: PredictWeatherRequest) -> None:
    """validate_request verifies incoming request headers and historical observation requirements."""
    if not request.node_id or not request.node_id.strip():
        raise ValidationError("node_id is required")

    if request.mode not in (PREDICTION_MODE_NEXT_24_HOURS, PREDICTION_MODE_NEXT_10_DAYS):
        raise ValidationError(f"unsupported prediction mode: {request.mode}")

    if not request.HasField("issue_time") or request.issue_time.seconds <= 0:
        raise ValidationError("valid issue_time is required")

    total_observations = len(request.history)
    if total_observations == 0:
        raise ValidationError("history observations cannot be empty")

    missing_temp_count = sum(1 for obs in request.history if not obs.HasField("temperature_c"))
    missing_hum_count = sum(1 for obs in request.history if not obs.HasField("humidity_pct"))

    if (missing_temp_count / total_observations) > 0.30:
        raise ValidationError("temperature_c exceeds maximum allowed missing ratio of 30 percent")

    if (missing_hum_count / total_observations) > 0.30:
        raise ValidationError("humidity_pct exceeds maximum allowed missing ratio of 30 percent")

    for obs_index, obs in enumerate(request.history):
        validate_observation_bounds(obs, obs_index)
