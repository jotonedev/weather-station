from collections.abc import Sequence
from datetime import UTC, datetime, timedelta

import numpy as np
import pandas as pd
from weather.inference.v1.inference_pb2 import TimeSeriesObservation

from inference.validator import ValidationError, trim_early_outage

# Optional columns enter the model only if they are not too sparse. Their
# effective missing ratio (absent or non-finite after validation) must be at or
# below this threshold.
OPTIONAL_COLUMN_THRESHOLD = 0.50

# Maximum consecutive missing samples bridged with linear interpolation
# (measured from each side of an interior gap, so the effective bridge is up to
# twice these values). Longer gaps keep a constant plateau in the middle instead
# of drawing one long trend across the outage.
MAX_GAP_HOURLY = 12
MAX_GAP_DAILY = 3


def floor_to_hour(dt: datetime) -> datetime:
    """floor_to_hour rounds a datetime down to the top of the hour in UTC."""
    return dt.replace(minute=0, second=0, microsecond=0)


def sort_observations(observations: Sequence[TimeSeriesObservation]) -> list[TimeSeriesObservation]:
    """sort_observations sorts observations chronologically by timestamp."""
    return sorted(
        observations,
        key=lambda item: item.timestamp.seconds + (item.timestamp.nanos * 1e-9),
    )


def _filter_and_trim_history(
    sorted_obs: list[TimeSeriesObservation],
    issue_time: datetime | None,
    step: timedelta,
) -> list[TimeSeriesObservation]:
    if issue_time is None:
        return sorted_obs
    step_seconds = int(step.total_seconds())
    if step >= timedelta(days=1):
        issue_sec = int(issue_time.timestamp())
        completed = [obs for obs in sorted_obs if obs.timestamp.seconds <= issue_sec]
        if completed:
            sorted_obs = completed
        return trim_early_outage(sorted_obs, issue_sec, step_seconds)

    floored_issue = floor_to_hour(issue_time)
    floored_issue_sec = int(floored_issue.timestamp())
    completed = [obs for obs in sorted_obs if obs.timestamp.seconds < floored_issue_sec]
    if completed:
        sorted_obs = completed
    return trim_early_outage(sorted_obs, floored_issue_sec, step_seconds)


def _compute_grid_bounds(
    sorted_obs: list[TimeSeriesObservation],
    issue_time: datetime | None,
    step: timedelta,
) -> tuple[datetime, datetime]:
    is_daily = step >= timedelta(days=1)
    first_dt = sorted_obs[0].timestamp.ToDatetime().replace(tzinfo=UTC)
    if issue_time is not None and not is_daily:
        first_dt = floor_to_hour(first_dt)

    if issue_time is not None:
        last_obs_dt = sorted_obs[-1].timestamp.ToDatetime().replace(tzinfo=UTC)
        if is_daily:
            expected_last_dt = issue_time.replace(hour=0, minute=0, second=0, microsecond=0)
        else:
            expected_last_dt = floor_to_hour(issue_time) - step
        grid_end = max(last_obs_dt, expected_last_dt)
    else:
        grid_end = sorted_obs[-1].timestamp.ToDatetime().replace(tzinfo=UTC)

    return first_dt, grid_end


def _observation_to_record(ts: datetime, obs: TimeSeriesObservation | None) -> dict[str, object]:
    if obs is None:
        return {
            "timestamp": ts,
            "temperature_c": np.nan,
            "min_temperature_c": np.nan,
            "max_temperature_c": np.nan,
            "humidity_pct": np.nan,
            "min_humidity_pct": np.nan,
            "max_humidity_pct": np.nan,
            "pressure_hpa": np.nan,
            "min_pressure_hpa": np.nan,
            "max_pressure_hpa": np.nan,
            "rainfall_mm": np.nan,
            "wind_speed_mps": np.nan,
            "wind_direction_deg": np.nan,
        }

    temp = obs.temperature_c if obs.HasField("temperature_c") else np.nan
    hum = obs.humidity_pct if obs.HasField("humidity_pct") else np.nan
    pressure = obs.pressure_hpa if obs.HasField("pressure_hpa") else np.nan

    return {
        "timestamp": ts,
        "temperature_c": temp,
        "min_temperature_c": obs.min_temperature_c if obs.HasField("min_temperature_c") else temp,
        "max_temperature_c": obs.max_temperature_c if obs.HasField("max_temperature_c") else temp,
        "humidity_pct": hum,
        "min_humidity_pct": obs.min_humidity_pct if obs.HasField("min_humidity_pct") else hum,
        "max_humidity_pct": obs.max_humidity_pct if obs.HasField("max_humidity_pct") else hum,
        "pressure_hpa": pressure,
        "min_pressure_hpa": obs.min_pressure_hpa if obs.HasField("min_pressure_hpa") else pressure,
        "max_pressure_hpa": obs.max_pressure_hpa if obs.HasField("max_pressure_hpa") else pressure,
        "rainfall_mm": obs.rainfall_mm if obs.HasField("rainfall_mm") else np.nan,
        "wind_speed_mps": obs.wind_speed_mps if obs.HasField("wind_speed_mps") else np.nan,
        "wind_direction_deg": obs.wind_direction_deg if obs.HasField("wind_direction_deg") else np.nan,
    }


def extract_observation_dataframe(
    observations: list[TimeSeriesObservation],
    step: timedelta = timedelta(hours=1),
    issue_time: datetime | None = None,
) -> pd.DataFrame:
    """extract_observation_dataframe extracts observation metrics into a regular time grid dataframe."""
    if not observations:
        return pd.DataFrame()

    sorted_obs = sort_observations(observations)
    sorted_obs = _filter_and_trim_history(sorted_obs, issue_time, step)
    first_dt, grid_end = _compute_grid_bounds(sorted_obs, issue_time, step)

    grid_timestamps: list[datetime] = []
    current = first_dt
    while current <= grid_end:
        grid_timestamps.append(current)
        current += step

    step_seconds = int(step.total_seconds())
    obs_by_index: dict[int, TimeSeriesObservation] = {}
    for obs in sorted_obs:
        obs_dt = obs.timestamp.ToDatetime().replace(tzinfo=UTC)
        idx = round((obs_dt - first_dt).total_seconds() / step_seconds)
        if 0 <= idx < len(grid_timestamps):
            obs_by_index[idx] = obs

    grid_records = [_observation_to_record(ts, obs_by_index.get(idx)) for idx, ts in enumerate(grid_timestamps)]
    df = pd.DataFrame(grid_records)
    # Normalize NaN and infinities to a single missing representation so that
    # imputation and sparsity checks treat them uniformly.
    return df.replace([np.inf, -np.inf], np.nan)


def _select_columns(df: pd.DataFrame) -> list[str]:
    total_rows = len(df)
    surviving_columns = [
        "temperature_c",
        "min_temperature_c",
        "max_temperature_c",
        "humidity_pct",
        "min_humidity_pct",
        "max_humidity_pct",
    ]

    optional_candidates = [
        "rainfall_mm",
        "pressure_hpa",
        "min_pressure_hpa",
        "max_pressure_hpa",
        "wind_speed_mps",
        "wind_direction_deg",
    ]
    for column_name in optional_candidates:
        if column_name in df.columns:
            missing_ratio = df[column_name].isna().sum() / total_rows
            if missing_ratio <= OPTIONAL_COLUMN_THRESHOLD:
                surviving_columns.append(column_name)
    return surviving_columns


def _impute_columns(clean_df: pd.DataFrame, df: pd.DataFrame, surviving_columns: list[str], max_gap: int) -> None:
    # Rainfall is episodic, so missing periods during downtime are imputed with 0.0
    if "rainfall_mm" in surviving_columns:
        clean_df["rainfall_mm"] = clean_df["rainfall_mm"].fillna(0.0)

    # For continuous weather metrics, use linear interpolation for short interior gaps
    # and diurnal / historical averages for long gaps
    has_timestamps = "timestamp" in df.columns
    hours = df["timestamp"].dt.hour if has_timestamps else None

    for col in surviving_columns:
        if col == "rainfall_mm":
            continue

        if hours is not None:
            diurnal_profile = df.groupby(hours)[col].transform("mean")
            col_mean = clean_df[col].mean()
            diurnal_profile = diurnal_profile.fillna(col_mean)
        else:
            diurnal_profile = clean_df[col].mean()

        clean_df[col] = clean_df[col].interpolate(
            method="linear",
            limit=max_gap,
            limit_direction="both",
            limit_area="inside",
        )
        clean_df[col] = clean_df[col].ffill(limit=2)
        clean_df[col] = clean_df[col].fillna(diurnal_profile)
        clean_df[col] = clean_df[col].bfill().ffill()


def _add_derived_features(clean_df: pd.DataFrame, surviving_columns: list[str]) -> None:
    # Derived meteorological features
    # 1. Diurnal temperature range (proxy for clear sky / radiative cooling)
    clean_df["temp_range_c"] = np.maximum(0.0, clean_df["max_temperature_c"] - clean_df["min_temperature_c"])
    surviving_columns.append("temp_range_c")

    # 2. Dew point calculation using Magnus-Tetens formula: bounds minimum night temperature
    # gamma(T, RH) = (b * T) / (c + T) + ln(RH / 100)
    # T_dew = (c * gamma) / (b - gamma), with b=17.625, c=243.04
    temp_arr = clean_df["temperature_c"].to_numpy(dtype=float)
    hum_clamped = np.clip(clean_df["humidity_pct"].to_numpy(dtype=float), 1.0, 100.0)
    b_const, c_const = 17.625, 243.04
    gamma = (b_const * temp_arr) / (c_const + temp_arr) + np.log(hum_clamped / 100.0)
    dew_point = (c_const * gamma) / (b_const - gamma)
    clean_df["dew_point_c"] = dew_point
    surviving_columns.append("dew_point_c")

    # 3. Pressure tendency (change in pressure)
    if "pressure_hpa" in surviving_columns:
        press_vals = clean_df["pressure_hpa"].to_numpy(dtype=float)
        press_diff = np.diff(press_vals, prepend=press_vals[0])
        clean_df["pressure_tendency_hpa"] = press_diff
        surviving_columns.append("pressure_tendency_hpa")

    # 4. Wind vector components u (eastward) and v (northward) in m/s
    if "wind_direction_deg" in surviving_columns:
        rad = np.radians(clean_df["wind_direction_deg"].to_numpy(dtype=float))
        speed = clean_df["wind_speed_mps"].to_numpy(dtype=float) if "wind_speed_mps" in surviving_columns else 1.0
        # Meteorological direction: angle wind is blowing FROM.
        # u = -speed * sin(dir), v = -speed * cos(dir)
        clean_df["wind_u"] = -speed * np.sin(rad)
        clean_df["wind_v"] = -speed * np.cos(rad)
        surviving_columns.extend(["wind_u", "wind_v"])


def filter_and_impute_columns(df: pd.DataFrame, max_gap: int = MAX_GAP_HOURLY) -> tuple[pd.DataFrame, list[str]]:
    """filter_and_impute_columns drops sparse optional columns and imputes missing values."""
    surviving_columns = _select_columns(df)
    clean_df = df[surviving_columns].copy()

    _impute_columns(clean_df, df, surviving_columns, max_gap)

    if not np.isfinite(clean_df.to_numpy(dtype=float)).all():
        raise ValidationError("history contains unresolvable missing values after imputation")

    _add_derived_features(clean_df, surviving_columns)
    return clean_df, surviving_columns


def generate_hourly_future_timestamps(issue_time: datetime, horizon_hours: int) -> list[datetime]:
    """generate_hourly_future_timestamps builds the future hourly timestamp sequence."""
    return [issue_time + timedelta(hours=step) for step in range(1, horizon_hours + 1)]


def generate_daily_future_timestamps(issue_time: datetime, horizon_days: int) -> list[datetime]:
    """generate_daily_future_timestamps builds the future daily timestamp sequence."""
    return [issue_time + timedelta(days=step) for step in range(1, horizon_days + 1)]


def calculate_temporal_covariates(
    timestamps: list[datetime],
    latitude: float | None = None,
) -> dict[str, np.ndarray]:
    """calculate_temporal_covariates computes cyclical diurnal, seasonal, and astronomical solar signals."""
    hours = np.array([dt.hour + (dt.minute / 60.0) for dt in timestamps], dtype=np.float32)
    days_of_year = np.array([dt.timetuple().tm_yday for dt in timestamps], dtype=np.float32)

    hour_angle = (2.0 * np.pi * hours) / 24.0
    doy_angle = (2.0 * np.pi * days_of_year) / 365.25

    # Solar declination angle delta (Spencer / Cooper approximation)
    solar_declination = 0.409 * np.sin(doy_angle - 1.39)

    lat_rad = np.radians(latitude if latitude is not None else 0.0)
    # Sunset hour angle: cos(omega_s) = -tan(lat) * tan(delta)
    # Clipped to [-1, 1] for polar day/night handling
    cos_omega_s = np.clip(-np.tan(lat_rad) * np.tan(solar_declination), -1.0, 1.0)
    omega_s = np.arccos(cos_omega_s)
    # Daylight fraction of 24 hours: omega_s / pi
    daylight_fraction = (omega_s / np.pi).astype(np.float32)

    return {
        "hour_sin": np.sin(hour_angle).astype(np.float32),
        "hour_cos": np.cos(hour_angle).astype(np.float32),
        "doy_sin": np.sin(doy_angle).astype(np.float32),
        "doy_cos": np.cos(doy_angle).astype(np.float32),
        "solar_declination": solar_declination.astype(np.float32),
        "daylight_fraction": daylight_fraction,
    }
