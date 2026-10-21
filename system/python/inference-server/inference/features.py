from collections.abc import Sequence
from datetime import UTC, datetime, timedelta

import numpy as np
import pandas as pd
from weather.inference.v1.inference_pb2 import TimeSeriesObservation

from inference.validator import ValidationError

# Optional columns enter the model only if they are not too sparse. Their
# effective missing ratio (absent or non-finite after validation) must be at or
# below this threshold.
OPTIONAL_COLUMN_THRESHOLD = 0.30

# Maximum consecutive missing samples bridged with linear interpolation
# (measured from each side of an interior gap, so the effective bridge is up to
# twice these values). Longer gaps keep a constant plateau in the middle instead
# of drawing one long trend across the outage.
MAX_GAP_HOURLY = 12
MAX_GAP_DAILY = 3


def sort_observations(observations: Sequence[TimeSeriesObservation]) -> list[TimeSeriesObservation]:
    """sort_observations sorts observations chronologically by timestamp."""
    return sorted(
        observations,
        key=lambda item: item.timestamp.seconds + (item.timestamp.nanos * 1e-9),
    )


def extract_observation_dataframe(observations: list[TimeSeriesObservation]) -> pd.DataFrame:
    """extract_observation_dataframe extracts observation metrics into a pandas dataframe."""
    records = []
    for obs in observations:
        dt = obs.timestamp.ToDatetime().replace(tzinfo=UTC)
        temp = obs.temperature_c if obs.HasField("temperature_c") else np.nan
        min_temp = obs.min_temperature_c if obs.HasField("min_temperature_c") else temp
        max_temp = obs.max_temperature_c if obs.HasField("max_temperature_c") else temp

        hum = obs.humidity_pct if obs.HasField("humidity_pct") else np.nan
        min_hum = obs.min_humidity_pct if obs.HasField("min_humidity_pct") else hum
        max_hum = obs.max_humidity_pct if obs.HasField("max_humidity_pct") else hum

        pressure = obs.pressure_hpa if obs.HasField("pressure_hpa") else np.nan
        min_press = obs.min_pressure_hpa if obs.HasField("min_pressure_hpa") else pressure
        max_press = obs.max_pressure_hpa if obs.HasField("max_pressure_hpa") else pressure

        record = {
            "timestamp": dt,
            "temperature_c": temp,
            "min_temperature_c": min_temp,
            "max_temperature_c": max_temp,
            "humidity_pct": hum,
            "min_humidity_pct": min_hum,
            "max_humidity_pct": max_hum,
            "pressure_hpa": pressure,
            "min_pressure_hpa": min_press,
            "max_pressure_hpa": max_press,
            "rainfall_mm": obs.rainfall_mm if obs.HasField("rainfall_mm") else np.nan,
            "wind_speed_mps": obs.wind_speed_mps if obs.HasField("wind_speed_mps") else np.nan,
            "wind_direction_deg": obs.wind_direction_deg if obs.HasField("wind_direction_deg") else np.nan,
        }
        records.append(record)

    df = pd.DataFrame(records)
    # Normalize NaN and infinities to a single missing representation so that
    # imputation and sparsity checks treat them uniformly.
    return df.replace([np.inf, -np.inf], np.nan)


def filter_and_impute_columns(df: pd.DataFrame, max_gap: int = MAX_GAP_HOURLY) -> tuple[pd.DataFrame, list[str]]:
    """filter_and_impute_columns drops sparse optional columns and imputes missing values."""
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

    clean_df = df[surviving_columns].copy()
    clean_df = clean_df.interpolate(
        method="linear",
        limit=max_gap,
        limit_direction="both",
        limit_area="inside",
    )
    clean_df = clean_df.bfill().ffill()

    if not np.isfinite(clean_df.to_numpy(dtype=float)).all():
        raise ValidationError("history contains unresolvable missing values after imputation")

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
