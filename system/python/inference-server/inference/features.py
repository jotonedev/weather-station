from collections.abc import Sequence
from datetime import UTC, datetime, timedelta

import numpy as np
import pandas as pd
from weather.inference.v1.inference_pb2 import (
    PREDICTION_MODE_NEXT_10_DAYS,
    PREDICTION_MODE_NEXT_24_HOURS,
    TimeSeriesObservation,
)

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
        record = {
            "timestamp": dt,
            "temperature_c": obs.temperature_c if obs.HasField("temperature_c") else np.nan,
            "humidity_pct": obs.humidity_pct if obs.HasField("humidity_pct") else np.nan,
            "rainfall_mm": obs.rainfall_mm if obs.HasField("rainfall_mm") else np.nan,
            "pressure_hpa": obs.pressure_hpa if obs.HasField("pressure_hpa") else np.nan,
        }
        records.append(record)

    df = pd.DataFrame(records)
    # Normalize NaN and infinities to a single missing representation so that
    # imputation and sparsity checks treat them uniformly.
    return df.replace([np.inf, -np.inf], np.nan)


def filter_and_impute_columns(df: pd.DataFrame, max_gap: int = MAX_GAP_HOURLY) -> tuple[pd.DataFrame, list[str]]:
    """filter_and_impute_columns drops sparse optional columns and imputes missing values."""
    total_rows = len(df)
    surviving_columns = ["temperature_c", "humidity_pct"]

    for column_name in ("rainfall_mm", "pressure_hpa"):
        if column_name in df.columns:
            missing_ratio = df[column_name].isna().sum() / total_rows
            if missing_ratio <= OPTIONAL_COLUMN_THRESHOLD:
                surviving_columns.append(column_name)

    clean_df = df[surviving_columns].copy()
    # Interpolate interior gaps linearly up to max_gap from each edge; gaps far
    # longer than the cap keep a constant plateau in the middle rather than a
    # single straight line spanning the outage.
    clean_df = clean_df.interpolate(
        method="linear",
        limit=max_gap,
        limit_direction="both",
        limit_area="inside",
    )
    clean_df = clean_df.bfill().ffill()

    # Required columns must never be silently zero-filled: the validator already
    # rejects histories whose required columns are too sparse, and non-finite
    # values are normalized to NaN by the extractor, so any NaN left here
    # indicates an unexpected imputation failure.
    if not np.isfinite(clean_df.to_numpy(dtype=float)).all():
        raise ValidationError("history contains unresolvable missing values after imputation")

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
