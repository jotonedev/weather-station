from collections.abc import Sequence
from datetime import UTC, datetime, timedelta

import numpy as np
import pandas as pd
from weather.inference.v1.inference_pb2 import TimeSeriesObservation

from inference.validator import DAILY_REQUIRED_FIELDS, HOURLY_REQUIRED_FIELDS, trim_early_outage


def floor_to_hour(dt: datetime) -> datetime:
    """floor_to_hour rounds a datetime down to the top of the hour in UTC."""
    return dt.replace(minute=0, second=0, microsecond=0)


def floor_to_day(dt: datetime) -> datetime:
    """floor_to_day rounds a datetime down to 00:00:00 UTC."""
    return dt.replace(hour=0, minute=0, second=0, microsecond=0)


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
    is_daily = step >= timedelta(days=1)
    floored_issue = floor_to_day(issue_time) if is_daily else floor_to_hour(issue_time)
    floored_issue_sec = int(floored_issue.timestamp())
    completed = [obs for obs in sorted_obs if obs.timestamp.seconds < floored_issue_sec]
    if not completed:
        return []
    required_fields = DAILY_REQUIRED_FIELDS if is_daily else HOURLY_REQUIRED_FIELDS
    return trim_early_outage(completed, floored_issue_sec, step_seconds, required_fields=required_fields)


def _compute_grid_bounds(
    sorted_obs: list[TimeSeriesObservation],
    issue_time: datetime | None,
    step: timedelta,
) -> tuple[datetime, datetime]:
    is_daily = step >= timedelta(days=1)
    first_dt = sorted_obs[0].timestamp.ToDatetime().replace(tzinfo=UTC)
    if issue_time is not None:
        first_dt = floor_to_day(first_dt) if is_daily else floor_to_hour(first_dt)
        last_obs_dt = sorted_obs[-1].timestamp.ToDatetime().replace(tzinfo=UTC)
        floored_issue = floor_to_day(issue_time) if is_daily else floor_to_hour(issue_time)
        expected_last_dt = floored_issue - step
        grid_end = max(last_obs_dt, expected_last_dt)
    else:
        grid_end = sorted_obs[-1].timestamp.ToDatetime().replace(tzinfo=UTC)

    return first_dt, grid_end


def _extract_wind_components(obs: TimeSeriesObservation, wind_speed: float, wind_dir: float) -> tuple[float, float]:
    """_extract_wind_components extracts (u, v) wind vectors or derives them from speed and direction."""
    has_u = obs.HasField("wind_u_mps")
    has_v = obs.HasField("wind_v_mps")
    if has_u and has_v:
        return float(obs.wind_u_mps), float(obs.wind_v_mps)

    if np.isfinite(wind_speed) and np.isfinite(wind_dir):
        rad = float(np.radians(wind_dir))
        u_val = float(obs.wind_u_mps) if has_u else float(-wind_speed * np.sin(rad))
        v_val = float(obs.wind_v_mps) if has_v else float(-wind_speed * np.cos(rad))
        return u_val, v_val

    return (
        float(obs.wind_u_mps) if has_u else np.nan,
        float(obs.wind_v_mps) if has_v else np.nan,
    )


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
            "wind_gust_mps": np.nan,
            "wind_u_mps": np.nan,
            "wind_v_mps": np.nan,
            "rain_rate_max_mmh": np.nan,
        }

    temp = obs.temperature_c if obs.HasField("temperature_c") else np.nan
    hum = obs.humidity_pct if obs.HasField("humidity_pct") else np.nan
    pressure = obs.pressure_hpa if obs.HasField("pressure_hpa") else np.nan
    wind_speed = obs.wind_speed_mps if obs.HasField("wind_speed_mps") else np.nan
    wind_dir = obs.wind_direction_deg if obs.HasField("wind_direction_deg") else np.nan
    wind_u, wind_v = _extract_wind_components(obs, wind_speed, wind_dir)

    return {
        "timestamp": ts,
        "temperature_c": temp,
        "min_temperature_c": obs.min_temperature_c if obs.HasField("min_temperature_c") else np.nan,
        "max_temperature_c": obs.max_temperature_c if obs.HasField("max_temperature_c") else np.nan,
        "humidity_pct": hum,
        "min_humidity_pct": obs.min_humidity_pct if obs.HasField("min_humidity_pct") else np.nan,
        "max_humidity_pct": obs.max_humidity_pct if obs.HasField("max_humidity_pct") else np.nan,
        "pressure_hpa": pressure,
        "min_pressure_hpa": obs.min_pressure_hpa if obs.HasField("min_pressure_hpa") else np.nan,
        "max_pressure_hpa": obs.max_pressure_hpa if obs.HasField("max_pressure_hpa") else np.nan,
        "rainfall_mm": obs.rainfall_mm if obs.HasField("rainfall_mm") else np.nan,
        "wind_speed_mps": wind_speed,
        "wind_direction_deg": wind_dir,
        "wind_gust_mps": obs.max_wind_speed_mps if obs.HasField("max_wind_speed_mps") else np.nan,
        "wind_u_mps": wind_u,
        "wind_v_mps": wind_v,
        "rain_rate_max_mmh": obs.max_rain_rate_mmh if obs.HasField("max_rain_rate_mmh") else np.nan,
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
    if not sorted_obs:
        return pd.DataFrame()
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
    # short-gap interpolation and sparsity checks treat them uniformly.
    return df.replace([np.inf, -np.inf], np.nan)


def generate_hourly_future_timestamps(issue_time: datetime, horizon_hours: int) -> list[datetime]:
    """generate_hourly_future_timestamps builds future hourly bucket start timestamps from issue_time."""
    floored = floor_to_hour(issue_time)
    return [floored + timedelta(hours=step) for step in range(horizon_hours)]


def generate_daily_future_timestamps(issue_time: datetime, horizon_days: int) -> list[datetime]:
    """generate_daily_future_timestamps builds future daily bucket start timestamps from 00:00 UTC."""
    floored = floor_to_day(issue_time)
    return [floored + timedelta(days=step) for step in range(horizon_days)]
