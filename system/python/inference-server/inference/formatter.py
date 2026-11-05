from collections.abc import Sequence

import numpy as np
from google.protobuf.timestamp_pb2 import Timestamp
from weather.database.v1.prediction_pb2 import DailyPrediction, HourlyPrediction

from inference.backends.base import QUANTILE_LEVELS
from inference.probability import (
    RAIN_THRESHOLD_DAILY_MM,
    RAIN_THRESHOLD_HOURLY_MM,
    exceedance_probability,
    quantile_at,
)


def _extract_clipped_triplet(
    curve: np.ndarray,
    levels: Sequence[float],
    clip_min: float,
    clip_max: float,
) -> tuple[float, float, float]:
    """_extract_clipped_triplet returns clipped (p10, p50, p90) values for one variate step."""
    p10 = float(np.clip(quantile_at(curve, levels, 0.1), clip_min, clip_max))
    p50 = float(np.clip(quantile_at(curve, levels, 0.5), clip_min, clip_max))
    p90 = float(np.clip(quantile_at(curve, levels, 0.9), clip_min, clip_max))
    return p10, p50, p90


def _extract_rainfall_stats(
    curve: np.ndarray,
    levels: Sequence[float],
    threshold_mm: float,
) -> tuple[float, float, float]:
    """_extract_rainfall_stats returns (median_mm, p90_mm, exceedance_probability)."""
    rain_prob = float(np.clip(exceedance_probability(curve, levels, threshold_mm), 0.0, 1.0))
    median_mm = float(max(0.0, quantile_at(curve, levels, 0.5))) if rain_prob > 0.0 else 0.0
    p90_mm = float(max(0.0, quantile_at(curve, levels, 0.9)))
    return median_mm, p90_mm, rain_prob


def _populate_hourly_step(
    prediction: HourlyPrediction,
    var_indices: dict[str, int],
    quantiles: np.ndarray,
    step_idx: int,
    quantile_levels: Sequence[float],
) -> None:
    if "temperature_c" in var_indices:
        curve = quantiles[var_indices["temperature_c"], step_idx]
        p10, p50, p90 = _extract_clipped_triplet(curve, quantile_levels, -90.0, 70.0)
        prediction.temperature_c = p50
        prediction.temperature_c_p10 = p10
        prediction.temperature_c_p90 = p90

    if "humidity_pct" in var_indices:
        curve = quantiles[var_indices["humidity_pct"], step_idx]
        p10, p50, p90 = _extract_clipped_triplet(curve, quantile_levels, 0.0, 100.0)
        prediction.humidity_pct = p50
        prediction.humidity_pct_p10 = p10
        prediction.humidity_pct_p90 = p90

    if "rainfall_mm" in var_indices:
        curve = quantiles[var_indices["rainfall_mm"], step_idx]
        expected_mm, p90_mm, rain_prob = _extract_rainfall_stats(curve, quantile_levels, RAIN_THRESHOLD_HOURLY_MM)
        prediction.rainfall_mm = expected_mm
        prediction.rainfall_mm_p90 = p90_mm
        prediction.rain_probability = rain_prob

    if "pressure_hpa" in var_indices:
        curve = quantiles[var_indices["pressure_hpa"], step_idx]
        p10, p50, p90 = _extract_clipped_triplet(curve, quantile_levels, 300.0, 1100.0)
        prediction.pressure_hpa = p50
        prediction.pressure_hpa_p10 = p10
        prediction.pressure_hpa_p90 = p90


def format_hourly_predictions(
    node_id: str,
    issue_time: Timestamp,
    horizon_hours: int,
    quantiles: np.ndarray,
    target_columns: list[str],
    quantile_levels: Sequence[float] = QUANTILE_LEVELS,
) -> list[HourlyPrediction]:
    """format_hourly_predictions converts model quantiles into HourlyPrediction records."""
    var_indices = {col: idx for idx, col in enumerate(target_columns)}
    predictions: list[HourlyPrediction] = []
    total_steps = min(horizon_hours, quantiles.shape[1])

    for step_idx in range(total_steps):
        prediction = HourlyPrediction(
            node_id=node_id,
            issue_time=issue_time,
            horizon_hours=step_idx + 1,
        )
        _populate_hourly_step(prediction, var_indices, quantiles, step_idx, quantile_levels)
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
    quantile_levels: Sequence[float],
) -> tuple[float | None, float | None, float | None, float | None]:
    """_extract_bounded_range retrieves and orders clipped min/max medians and outer p10/p90 bounds."""
    min_val, max_val = None, None
    min_p10, max_p90 = None, None
    if min_col in var_indices:
        curve = quantiles[var_indices[min_col], step_idx]
        min_p10, min_val, _ = _extract_clipped_triplet(curve, quantile_levels, clip_min, clip_max)
    if max_col in var_indices:
        curve = quantiles[var_indices[max_col], step_idx]
        _, max_val, max_p90 = _extract_clipped_triplet(curve, quantile_levels, clip_min, clip_max)
    if min_val is not None and max_val is not None and min_val > max_val:
        min_val, max_val = max_val, min_val
    if min_p10 is not None and min_val is not None:
        min_p10 = min(min_p10, min_val)
    if max_p90 is not None and max_val is not None:
        max_p90 = max(max_p90, max_val)
    return min_val, max_val, min_p10, max_p90


def _populate_daily_step(
    prediction: DailyPrediction,
    var_indices: dict[str, int],
    quantiles: np.ndarray,
    step_idx: int,
    quantile_levels: Sequence[float],
) -> None:
    min_temp, max_temp, min_temp_p10, max_temp_p90 = _extract_bounded_range(
        var_indices,
        quantiles,
        step_idx,
        "min_temperature_c",
        "max_temperature_c",
        -90.0,
        70.0,
        quantile_levels,
    )
    if min_temp is not None:
        prediction.min_temperature_c = min_temp
    if max_temp is not None:
        prediction.max_temperature_c = max_temp
    if min_temp_p10 is not None:
        prediction.min_temperature_c_p10 = min_temp_p10
    if max_temp_p90 is not None:
        prediction.max_temperature_c_p90 = max_temp_p90

    min_hum, max_hum, _, _ = _extract_bounded_range(
        var_indices,
        quantiles,
        step_idx,
        "min_humidity_pct",
        "max_humidity_pct",
        0.0,
        100.0,
        quantile_levels,
    )
    if min_hum is not None:
        prediction.min_humidity_pct = min_hum
    if max_hum is not None:
        prediction.max_humidity_pct = max_hum

    if "rainfall_mm" in var_indices:
        curve = quantiles[var_indices["rainfall_mm"], step_idx]
        expected_mm, p90_mm, rain_prob = _extract_rainfall_stats(curve, quantile_levels, RAIN_THRESHOLD_DAILY_MM)
        prediction.rainfall_mm = expected_mm
        prediction.rainfall_mm_p90 = p90_mm
        prediction.rain_probability = rain_prob

    if "pressure_hpa" in var_indices:
        curve = quantiles[var_indices["pressure_hpa"], step_idx]
        p10, p50, p90 = _extract_clipped_triplet(curve, quantile_levels, 300.0, 1100.0)
        prediction.pressure_hpa = p50
        prediction.pressure_hpa_p10 = p10
        prediction.pressure_hpa_p90 = p90


def format_daily_predictions(
    node_id: str,
    issue_time: Timestamp,
    horizon_days: int,
    quantiles: np.ndarray,
    target_columns: list[str],
    quantile_levels: Sequence[float] = QUANTILE_LEVELS,
) -> list[DailyPrediction]:
    """format_daily_predictions converts model quantiles into DailyPrediction records starting at day 0."""
    var_indices = {col: idx for idx, col in enumerate(target_columns)}
    predictions: list[DailyPrediction] = []
    total_steps = min(horizon_days, quantiles.shape[1])

    for step_idx in range(total_steps):
        prediction = DailyPrediction(
            node_id=node_id,
            issue_time=issue_time,
            horizon_days=step_idx,
        )
        _populate_daily_step(prediction, var_indices, quantiles, step_idx, quantile_levels)
        predictions.append(prediction)

    return predictions
