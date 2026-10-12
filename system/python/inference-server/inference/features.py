import math
from collections.abc import Sequence
from datetime import UTC, datetime, timedelta

import numpy as np
import pandas as pd
from weather.inference.v1.inference_pb2 import (
    PREDICTION_MODE_NEXT_10_DAYS,
    PREDICTION_MODE_NEXT_24_HOURS,
    TimeSeriesObservation,
)


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
        record = {
            "timestamp": dt,
            "temperature_c": obs.temperature_c if obs.HasField("temperature_c") else np.nan,
            "humidity_pct": obs.humidity_pct if obs.HasField("humidity_pct") else np.nan,
            "rainfall_mm": obs.rainfall_mm if obs.HasField("rainfall_mm") else np.nan,
            "pressure_hpa": obs.pressure_hpa if obs.HasField("pressure_hpa") else np.nan,
        }

        if obs.HasField("wind_speed_mps") and obs.HasField("wind_direction_deg"):
            rad = math.radians(obs.wind_direction_deg)
            record["wind_u"] = math.sin(rad)
            record["wind_v"] = math.cos(rad)
        else:
            record["wind_u"] = np.nan
            record["wind_v"] = np.nan

        records.append(record)

    return pd.DataFrame(records)


def filter_and_impute_columns(df: pd.DataFrame) -> tuple[pd.DataFrame, list[str]]:
    """filter_and_impute_columns drops sparse columns and imputes missing values."""
    total_rows = len(df)
    surviving_columns = ["temperature_c", "humidity_pct"]

    # Optional rainfall evaluation
    if "rainfall_mm" in df.columns:
        rain_missing = df["rainfall_mm"].isna().sum() / total_rows
        if rain_missing <= 0.30:
            surviving_columns.append("rainfall_mm")

    # Optional wind evaluation requiring both components
    if "wind_u" in df.columns and "wind_v" in df.columns:
        wind_u_missing = df["wind_u"].isna().sum() / total_rows
        wind_v_missing = df["wind_v"].isna().sum() / total_rows
        if wind_u_missing <= 0.30 and wind_v_missing <= 0.30:
            surviving_columns.extend(["wind_u", "wind_v"])

    # Interpolate and backfill / forward fill
    clean_df = df[surviving_columns].copy()
    clean_df = clean_df.interpolate(method="linear", limit_direction="both")
    clean_df = clean_df.bfill().ffill().fillna(0.0)

    return clean_df, surviving_columns


def generate_future_timestamps(issue_time: datetime, mode: int) -> list[datetime]:
    """generate_future_timestamps builds the future timestamp sequence for the forecast horizon."""
    if mode == PREDICTION_MODE_NEXT_24_HOURS:
        return [issue_time + timedelta(hours=step) for step in range(1, 25)]
    if mode == PREDICTION_MODE_NEXT_10_DAYS:
        return [issue_time + timedelta(days=step) for step in range(1, 11)]
    return []


def calculate_temporal_covariates(timestamps: list[datetime]) -> dict[str, np.ndarray]:
    """calculate_temporal_covariates computes cyclical diurnal and seasonal sine and cosine signals."""
    hours = np.array([dt.hour + (dt.minute / 60.0) for dt in timestamps], dtype=np.float32)
    days_of_year = np.array([dt.timetuple().tm_yday for dt in timestamps], dtype=np.float32)

    hour_angle = (2.0 * np.pi * hours) / 24.0
    doy_angle = (2.0 * np.pi * days_of_year) / 365.25

    return {
        "hour_sin": np.sin(hour_angle).astype(np.float32),
        "hour_cos": np.cos(hour_angle).astype(np.float32),
        "doy_sin": np.sin(doy_angle).astype(np.float32),
        "doy_cos": np.cos(doy_angle).astype(np.float32),
    }


def reconstruct_wind_direction(u: float, v: float) -> float:
    """reconstruct_wind_direction recovers directional degrees from orthogonal u and v vector components."""
    rad = math.atan2(u, v)
    deg = math.degrees(rad)
    return (deg + 360.0) % 360.0
