"""Target and past-covariate construction for hourly and daily weather forecasts."""

from dataclasses import dataclass

import numpy as np
import pandas as pd

OPTIONAL_COLUMN_THRESHOLD = 0.50
SHORT_GAP_LIMIT = 2

HOURLY_REQUIRED_TARGETS: tuple[str, ...] = (
    "temperature_c",
    "humidity_pct",
)
HOURLY_OPTIONAL_TARGETS: tuple[str, ...] = (
    "rainfall_mm",
    "pressure_hpa",
)
HOURLY_SENSOR_PAST_COVARIATES: tuple[str, ...] = (
    "min_temperature_c",
    "max_temperature_c",
    "min_humidity_pct",
    "max_humidity_pct",
    "min_pressure_hpa",
    "max_pressure_hpa",
    "wind_speed_mps",
    "wind_gust_mps",
    "wind_u_mps",
    "wind_v_mps",
    "rain_rate_max_mmh",
)

DAILY_REQUIRED_TARGETS: tuple[str, ...] = (
    "min_temperature_c",
    "max_temperature_c",
    "min_humidity_pct",
    "max_humidity_pct",
)
DAILY_OPTIONAL_TARGETS: tuple[str, ...] = (
    "rainfall_mm",
    "pressure_hpa",
)
DAILY_SENSOR_PAST_COVARIATES: tuple[str, ...] = (
    "temperature_c",
    "humidity_pct",
    "min_pressure_hpa",
    "max_pressure_hpa",
    "wind_speed_mps",
    "wind_gust_mps",
    "wind_u_mps",
    "wind_v_mps",
    "rain_rate_max_mmh",
)


@dataclass(frozen=True)
class PreparedContext:
    """PreparedContext holds the target matrix, target names, and past-only covariate series."""

    target: np.ndarray
    target_columns: list[str]
    past_covariates: dict[str, np.ndarray]


def _passes_density_threshold(series: pd.Series, threshold: float = OPTIONAL_COLUMN_THRESHOLD) -> bool:
    """_passes_density_threshold checks that a column's missing ratio does not exceed threshold."""
    if len(series) == 0:
        return False
    missing_ratio = float(series.isna().sum()) / float(len(series))
    return missing_ratio <= threshold


def _interpolate_short_gaps(series: pd.Series, column_name: str, max_gap: int = SHORT_GAP_LIMIT) -> pd.Series:
    """_interpolate_short_gaps bridges only 1-2 step interior gaps and leaves longer outages as NaN."""
    cleaned = series.astype(float).replace([np.inf, -np.inf], np.nan)
    if max_gap <= 0 or cleaned.isna().all():
        return cleaned
    if column_name in ("rainfall_mm", "rain_rate_max_mmh"):
        # Do not linearly ramp precipitation across missing steps; leave gaps for native masking.
        return cleaned
    return cleaned.interpolate(
        method="linear",
        limit=max_gap,
        limit_direction="forward",
        limit_area="inside",
    )


def _compute_moisture_features(temp_c: np.ndarray, hum_pct: np.ndarray) -> dict[str, np.ndarray]:
    """_compute_moisture_features derives dew point, dew point depression, and absolute humidity."""
    with np.errstate(invalid="ignore", divide="ignore"):
        hum_clamped = np.clip(hum_pct, 1.0, 100.0)
        b_const, c_const = 17.625, 243.04
        gamma = ((b_const * temp_c) / (c_const + temp_c)) + np.log(hum_clamped / 100.0)
        dew_point = (c_const * gamma) / (b_const - gamma)
        dew_depression = np.maximum(0.0, temp_c - dew_point)

        sat_vapor_hpa = 6.112 * np.exp((17.67 * temp_c) / (temp_c + 243.5))
        abs_humidity = (sat_vapor_hpa * hum_clamped * 2.1674) / (273.15 + temp_c)

    return {
        "dew_point_c": dew_point.astype(np.float32),
        "dew_point_depression_c": dew_depression.astype(np.float32),
        "absolute_humidity_gm3": abs_humidity.astype(np.float32),
    }


def _lag_difference(values: np.ndarray, lag: int) -> np.ndarray:
    """_lag_difference computes values[t] - values[max(0, t - lag)] while preserving NaNs."""
    n = values.size
    if n == 0:
        return values.astype(np.float32)
    indices = np.arange(n)
    lagged_indices = np.maximum(0, indices - lag)
    with np.errstate(invalid="ignore"):
        diff = values - values[lagged_indices]
    return diff.astype(np.float32)


def _rolling_rain_sum(rain: pd.Series, window: int) -> np.ndarray:
    """_rolling_rain_sum computes a trailing rolling rain total while keeping outage rows as NaN."""
    rolled = rain.rolling(window=window, min_periods=1).sum()
    rolled = rolled.where(rain.notna(), np.nan)
    return rolled.to_numpy(dtype=np.float32)


def _build_derived_covariates(
    interpolated: dict[str, pd.Series],
    is_daily: bool,
) -> dict[str, np.ndarray]:
    """_build_derived_covariates computes thermodynamic, barometric, and precipitation covariates."""
    derived: dict[str, np.ndarray] = {}

    if "min_temperature_c" in interpolated and "max_temperature_c" in interpolated:
        min_t = interpolated["min_temperature_c"].to_numpy(dtype=float)
        max_t = interpolated["max_temperature_c"].to_numpy(dtype=float)
        with np.errstate(invalid="ignore"):
            derived["temp_range_c"] = np.maximum(0.0, max_t - min_t).astype(np.float32)

    if "temperature_c" in interpolated and "humidity_pct" in interpolated:
        temp_arr = interpolated["temperature_c"].to_numpy(dtype=float)
        hum_arr = interpolated["humidity_pct"].to_numpy(dtype=float)
        derived.update(_compute_moisture_features(temp_arr, hum_arr))

    if "pressure_hpa" in interpolated and _passes_density_threshold(interpolated["pressure_hpa"]):
        press_arr = interpolated["pressure_hpa"].to_numpy(dtype=float)
        if is_daily:
            derived["pressure_tendency_24h_hpa"] = _lag_difference(press_arr, lag=1)
        else:
            derived["pressure_tendency_3h_hpa"] = _lag_difference(press_arr, lag=3)
            derived["pressure_tendency_24h_hpa"] = _lag_difference(press_arr, lag=24)

    if "rainfall_mm" in interpolated and _passes_density_threshold(interpolated["rainfall_mm"]):
        rain_series = interpolated["rainfall_mm"]
        if is_daily:
            derived["rain_sum_7d_mm"] = _rolling_rain_sum(rain_series, window=7)
        else:
            derived["rain_sum_24h_mm"] = _rolling_rain_sum(rain_series, window=24)

    return derived


def prepare_context_and_covariates(
    df: pd.DataFrame,
    is_daily: bool = False,
    short_gap_limit: int = SHORT_GAP_LIMIT,
) -> PreparedContext:
    """prepare_context_and_covariates builds target matrix and past covariates with short-gap interpolation."""
    required_targets = DAILY_REQUIRED_TARGETS if is_daily else HOURLY_REQUIRED_TARGETS
    optional_targets = DAILY_OPTIONAL_TARGETS if is_daily else HOURLY_OPTIONAL_TARGETS
    sensor_covariates = DAILY_SENSOR_PAST_COVARIATES if is_daily else HOURLY_SENSOR_PAST_COVARIATES

    interpolated: dict[str, pd.Series] = {}
    for col in df.columns:
        if col == "timestamp":
            continue
        interpolated[col] = _interpolate_short_gaps(df[col], col, max_gap=short_gap_limit)

    target_columns: list[str] = list(required_targets)
    for col in optional_targets:
        if col in interpolated and _passes_density_threshold(interpolated[col]):
            target_columns.append(col)

    target_rows = [interpolated[col].to_numpy(dtype=np.float32) for col in target_columns]
    target_matrix = np.stack(target_rows, axis=0) if target_rows else np.empty((0, len(df)), dtype=np.float32)

    past_covariates: dict[str, np.ndarray] = {}
    for col in sensor_covariates:
        if col in interpolated and _passes_density_threshold(interpolated[col]):
            past_covariates[col] = interpolated[col].to_numpy(dtype=np.float32)

    derived = _build_derived_covariates(interpolated, is_daily=is_daily)
    for name, arr in derived.items():
        if _passes_density_threshold(pd.Series(arr)):
            past_covariates[name] = arr

    return PreparedContext(
        target=target_matrix,
        target_columns=target_columns,
        past_covariates=past_covariates,
    )
