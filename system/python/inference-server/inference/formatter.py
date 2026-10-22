import numpy as np
from google.protobuf.timestamp_pb2 import Timestamp
from weather.database.v1.prediction_pb2 import DailyPrediction, HourlyPrediction


def compute_confidence(spread: float, scale: float) -> float:
    """compute_confidence translates prediction interval spread to a normalized 0 to 1 confidence."""
    confidence = 1.0 - (max(0.0, spread) / scale)
    return float(np.clip(confidence, 0.0, 1.0))


def format_hourly_predictions(
    node_id: str,
    issue_time: Timestamp,
    horizon_hours: int,
    quantiles: np.ndarray,
    target_columns: list[str],
) -> list[HourlyPrediction]:
    """format_hourly_predictions converts model quantiles into HourlyPrediction records."""
    var_indices = {col: idx for idx, col in enumerate(target_columns)}
    predictions = []
    total_steps = min(horizon_hours, quantiles.shape[1])

    for step_idx in range(total_steps):
        step_hour = step_idx + 1
        prediction = HourlyPrediction(
            node_id=node_id,
            issue_time=issue_time,
            horizon_hours=step_hour,
        )

        if "temperature_c" in var_indices:
            idx = var_indices["temperature_c"]
            prediction.temperature_c = float(np.clip(quantiles[idx, step_idx, 1], -90.0, 70.0))

        if "humidity_pct" in var_indices:
            idx = var_indices["humidity_pct"]
            prediction.humidity_pct = float(np.clip(quantiles[idx, step_idx, 1], 0.0, 100.0))

        if "rainfall_mm" in var_indices:
            idx = var_indices["rainfall_mm"]
            rain_med = float(max(0.0, quantiles[idx, step_idx, 1]))
            rain_spread = float(quantiles[idx, step_idx, 2] - quantiles[idx, step_idx, 0])
            prediction.rainfall_mm = rain_med
            prediction.rainfall_confidence = compute_confidence(rain_spread, 20.0)

        predictions.append(prediction)

    return predictions


def _extract_bounded_range(
    var_indices: dict[str, int],
    quantiles: np.ndarray,
    step_idx: int,
    min_col: str,
    max_col: str,
    clip_min: float,
    clip_max: float,
) -> tuple[float | None, float | None]:
    """_extract_bounded_range retrieves and orders clipped min/max quantiles."""
    min_val, max_val = None, None
    if min_col in var_indices:
        min_val = float(np.clip(quantiles[var_indices[min_col], step_idx, 1], clip_min, clip_max))
    if max_col in var_indices:
        max_val = float(np.clip(quantiles[var_indices[max_col], step_idx, 1], clip_min, clip_max))
    if min_val is not None and max_val is not None and min_val > max_val:
        min_val, max_val = max_val, min_val
    return min_val, max_val


def format_daily_predictions(
    node_id: str,
    issue_time: Timestamp,
    horizon_days: int,
    quantiles: np.ndarray,
    target_columns: list[str],
) -> list[DailyPrediction]:
    """format_daily_predictions converts model quantiles into DailyPrediction records."""
    var_indices = {col: idx for idx, col in enumerate(target_columns)}
    predictions = []
    total_steps = min(horizon_days, quantiles.shape[1])

    for step_idx in range(total_steps):
        prediction = DailyPrediction(
            node_id=node_id,
            issue_time=issue_time,
            horizon_days=step_idx + 1,
        )

        min_temp, max_temp = _extract_bounded_range(
            var_indices, quantiles, step_idx, "min_temperature_c", "max_temperature_c", -90.0, 70.0
        )
        if min_temp is not None:
            prediction.min_temperature_c = min_temp
        if max_temp is not None:
            prediction.max_temperature_c = max_temp

        min_hum, max_hum = _extract_bounded_range(
            var_indices, quantiles, step_idx, "min_humidity_pct", "max_humidity_pct", 0.0, 100.0
        )
        if min_hum is not None:
            prediction.min_humidity_pct = min_hum
        if max_hum is not None:
            prediction.max_humidity_pct = max_hum

        if "rainfall_mm" in var_indices:
            idx = var_indices["rainfall_mm"]
            rain_med = float(max(0.0, quantiles[idx, step_idx, 1]))
            rain_spread = float(quantiles[idx, step_idx, 2] - quantiles[idx, step_idx, 0])
            prediction.rainfall_mm = rain_med
            prediction.rainfall_confidence = compute_confidence(rain_spread, 20.0)

        predictions.append(prediction)

    return predictions
