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
    """_lag_difference computes values[t] - values[t - lag] for t >= lag and NaN for t < lag."""
    length = values.size
    if length == 0:
        return values.astype(np.float32)
    diff = np.full(length, np.nan, dtype=np.float32)
    if lag < length:
        with np.errstate(invalid="ignore"):
            diff[lag:] = (values[lag:] - values[:-lag]).astype(np.float32)
    return diff


def _rolling_rain_sum(rain: pd.Series, window: int) -> np.ndarray:
    """_rolling_rain_sum computes a trailing rolling rain total while keeping warmup and outage rows as NaN."""
    rolled = rain.rolling(window=window, min_periods=window).sum()
    rolled = rolled.where(rain.notna(), np.nan)
    return rolled.to_numpy(dtype=np.float32)


def _build_daily_target_series(
    interpolated: dict[str, pd.Series],
    col: str,
    steps_per_day: int,
) -> pd.Series:
    """_build_daily_target_series converts sub-daily buckets into trailing 24-hour daily target series."""
    series = interpolated[col]
    if steps_per_day <= 1 or len(series) == 0:
        return series
    observed_total = int(series.notna().sum())
    if observed_total == 0:
        return series
    min_obs = max(1, min(steps_per_day // 2, observed_total))

    if col in ("min_temperature_c", "min_humidity_pct"):
        rolled = series.rolling(window=steps_per_day, min_periods=min_obs).min()
    elif col in ("max_temperature_c", "max_humidity_pct"):
        rolled = series.rolling(window=steps_per_day, min_periods=min_obs).max()
    elif col == "rainfall_mm":
        rolled = series.rolling(window=steps_per_day, min_periods=min_obs).mean() * float(steps_per_day)
    elif col == "pressure_hpa":
        rolled = series.rolling(window=steps_per_day, min_periods=min_obs).mean()
    else:
        return series

    return rolled.where(series.notna(), np.nan)


def _build_derived_covariates(
    interpolated: dict[str, pd.Series],
    is_daily: bool,
    steps_per_day: int = 1,
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
            if steps_per_day >= 8:
                derived["pressure_tendency_3h_hpa"] = _lag_difference(press_arr, lag=max(1, steps_per_day // 8))
            derived["pressure_tendency_24h_hpa"] = _lag_difference(press_arr, lag=max(1, steps_per_day))
        else:
            derived["pressure_tendency_3h_hpa"] = _lag_difference(press_arr, lag=3)
            derived["pressure_tendency_24h_hpa"] = _lag_difference(press_arr, lag=24)

    if "rainfall_mm" in interpolated and _passes_density_threshold(interpolated["rainfall_mm"]):
        rain_series = interpolated["rainfall_mm"]
        rain_arr = rain_series.to_numpy(dtype=float)
        with np.errstate(invalid="ignore"):
            derived["rain_occurrence"] = np.where(
                np.isnan(rain_arr),
                np.nan,
                (rain_arr >= 0.2).astype(np.float32),
            ).astype(np.float32)
        if is_daily:
            derived["rain_sum_7d_mm"] = _rolling_rain_sum(rain_series, window=7 * max(1, steps_per_day))
        else:
            derived["rain_sum_24h_mm"] = _rolling_rain_sum(rain_series, window=24)

    return derived


def _collect_past_covariates(
    interpolated: dict[str, pd.Series],
    is_daily: bool,
    steps_per_day: int,
) -> dict[str, np.ndarray]:
    """_collect_past_covariates gathers sensor and derived past-only covariates meeting density rules."""
    sensor_covariates = DAILY_SENSOR_PAST_COVARIATES if is_daily else HOURLY_SENSOR_PAST_COVARIATES
    past_covariates: dict[str, np.ndarray] = {}
    for col in sensor_covariates:
        if col in interpolated and _passes_density_threshold(interpolated[col]):
            past_covariates[col] = interpolated[col].to_numpy(dtype=np.float32)

    if is_daily and steps_per_day > 1:
        for col in ("min_temperature_c", "max_temperature_c", "rainfall_mm", "pressure_hpa"):
            if col in interpolated and _passes_density_threshold(interpolated[col]):
                past_covariates[f"bucket_{col}"] = interpolated[col].to_numpy(dtype=np.float32)

    derived = _build_derived_covariates(interpolated, is_daily=is_daily, steps_per_day=steps_per_day)
    for name, arr in derived.items():
        if _passes_density_threshold(pd.Series(arr)):
            past_covariates[name] = arr

    return past_covariates


def prepare_context_and_covariates(
    df: pd.DataFrame,
    is_daily: bool = False,
    steps_per_day: int | None = None,
    short_gap_limit: int = SHORT_GAP_LIMIT,
) -> PreparedContext:
    """prepare_context_and_covariates builds target matrix and past covariates with short-gap interpolation."""
    effective_steps_per_day = (8 if is_daily else 1) if steps_per_day is None else steps_per_day
    required_targets = DAILY_REQUIRED_TARGETS if is_daily else HOURLY_REQUIRED_TARGETS
    optional_targets = DAILY_OPTIONAL_TARGETS if is_daily else HOURLY_OPTIONAL_TARGETS

    interpolated = {
        col: _interpolate_short_gaps(df[col], col, max_gap=short_gap_limit) for col in df.columns if col != "timestamp"
    }

    target_columns: list[str] = [col for col in required_targets if col in interpolated]
    for col in optional_targets:
        if col in interpolated and _passes_density_threshold(interpolated[col]):
            target_columns.append(col)

    if is_daily and effective_steps_per_day > 1:
        target_rows = [
            _build_daily_target_series(interpolated, col, effective_steps_per_day).to_numpy(dtype=np.float32)
            for col in target_columns
        ]
    else:
        target_rows = [interpolated[col].to_numpy(dtype=np.float32) for col in target_columns]
    target_matrix = np.stack(target_rows, axis=0) if target_rows else np.empty((0, len(df)), dtype=np.float32)
    past_covariates = _collect_past_covariates(interpolated, is_daily=is_daily, steps_per_day=effective_steps_per_day)

    return PreparedContext(
        target=target_matrix,
        target_columns=target_columns,
        past_covariates=past_covariates,
    )
