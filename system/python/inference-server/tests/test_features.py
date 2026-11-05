from datetime import UTC, datetime, timedelta

import numpy as np
from google.protobuf.timestamp_pb2 import Timestamp
from weather.inference.v1.inference_pb2 import TimeSeriesObservation

from inference.covariates import prepare_context_and_covariates
from inference.features import (
    extract_observation_dataframe,
    floor_to_day,
    floor_to_hour,
    generate_daily_future_timestamps,
    generate_hourly_future_timestamps,
    sort_observations,
)


def test_sort_observations() -> None:
    """Test_sort_observations verifies observations are sorted in ascending chronological order."""
    timestamp_first = Timestamp()
    timestamp_first.FromSeconds(100)
    timestamp_second = Timestamp()
    timestamp_second.FromSeconds(200)

    observation_one = TimeSeriesObservation(timestamp=timestamp_second)
    observation_two = TimeSeriesObservation(timestamp=timestamp_first)

    sorted_result = sort_observations([observation_one, observation_two])
    assert sorted_result[0].timestamp.seconds == 100
    assert sorted_result[1].timestamp.seconds == 200


def test_floor_helpers() -> None:
    """Test_floor_helpers verifies hourly and daily timestamp flooring."""
    dt = datetime(2026, 6, 15, 14, 37, 22, tzinfo=UTC)
    assert floor_to_hour(dt) == datetime(2026, 6, 15, 14, 0, 0, tzinfo=UTC)
    assert floor_to_day(dt) == datetime(2026, 6, 15, 0, 0, 0, tzinfo=UTC)


def test_wind_features_fallback_from_direction() -> None:
    """Test_wind_features_fallback_from_direction derives wind_u_mps and wind_v_mps when absent."""
    timestamp = Timestamp()
    timestamp.FromSeconds(1000)

    observations = [
        TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=20.0,
            humidity_pct=50.0,
            wind_speed_mps=5.0,
            wind_direction_deg=90.0,
        )
    ]

    dataframe = extract_observation_dataframe(observations)
    prepared = prepare_context_and_covariates(dataframe)

    assert "wind_speed_mps" in prepared.past_covariates
    assert "wind_u_mps" in prepared.past_covariates
    assert "wind_v_mps" in prepared.past_covariates
    assert np.isclose(float(prepared.past_covariates["wind_u_mps"][0]), -5.0)
    assert np.isclose(float(prepared.past_covariates["wind_v_mps"][0]), 0.0, atol=1e-5)


def test_column_filtering_sparse_column_dropped() -> None:
    """Test_column_filtering_sparse_column_dropped verifies dropping columns with missing ratio over 50 percent."""
    observations = []
    for step_index in range(10):
        timestamp = Timestamp()
        timestamp.FromSeconds(1000 + (step_index * 3600))
        observation = TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=20.0,
            humidity_pct=50.0,
        )
        # Supply rainfall only for 4 out of 10 observations (60 percent missing)
        if step_index < 4:
            observation.rainfall_mm = 1.2

        observations.append(observation)

    dataframe = extract_observation_dataframe(observations)
    prepared = prepare_context_and_covariates(dataframe)

    assert "rainfall_mm" not in prepared.target_columns
    assert "temperature_c" in prepared.target_columns
    assert "humidity_pct" in prepared.target_columns


def test_column_filtering_dense_column_retained() -> None:
    """Test_column_filtering_dense_column_retained verifies keeping columns with missing ratio under 50 percent."""
    observations = []
    for step_index in range(10):
        timestamp = Timestamp()
        timestamp.FromSeconds(1000 + (step_index * 3600))
        observation = TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=20.0,
            humidity_pct=50.0,
            pressure_hpa=1013.0,
        )
        if step_index < 6:
            observation.rainfall_mm = 0.5

        observations.append(observation)

    dataframe = extract_observation_dataframe(observations)
    prepared = prepare_context_and_covariates(dataframe)

    assert "rainfall_mm" in prepared.target_columns
    assert "pressure_hpa" in prepared.target_columns


def test_inf_normalized_to_missing_and_interpolated() -> None:
    """Test_inf_normalized_to_missing_and_interpolated ensures infinities are normalized and short gaps bridged."""
    observations = []
    for step_index in range(10):
        timestamp = Timestamp()
        timestamp.FromSeconds(1000 + (step_index * 3600))
        temperature = 20.0 + step_index
        if step_index in (2, 3):
            temperature = float("inf")
        observation = TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=temperature,
            humidity_pct=60.0,
            pressure_hpa=1013.0,
        )
        observations.append(observation)

    dataframe = extract_observation_dataframe(observations)
    assert dataframe["temperature_c"].iloc[2] != float("inf")
    assert dataframe["temperature_c"].isna().sum() == 2

    prepared = prepare_context_and_covariates(dataframe)
    values = prepared.target[0]
    assert np.isfinite(values).all()


def test_generate_future_timestamps_contiguous_with_past() -> None:
    """Test_generate_future_timestamps_contiguous_with_past checks bucket starts begin at floored issue_time."""
    issue_time = datetime(2026, 1, 1, 12, 35, tzinfo=UTC)

    hourly_steps = generate_hourly_future_timestamps(issue_time, 24)
    assert len(hourly_steps) == 24
    assert hourly_steps[0] == datetime(2026, 1, 1, 12, 0, tzinfo=UTC)
    assert hourly_steps[-1] == datetime(2026, 1, 2, 11, 0, tzinfo=UTC)

    daily_steps = generate_daily_future_timestamps(issue_time, 10)
    assert len(daily_steps) == 10
    assert daily_steps[0] == datetime(2026, 1, 1, 0, 0, tzinfo=UTC)
    assert daily_steps[-1] == datetime(2026, 1, 10, 0, 0, tzinfo=UTC)


def test_extract_observation_dataframe_reindexes_interior_gaps() -> None:
    """Test_extract_observation_dataframe_reindexes_interior_gaps ensures skipped buckets become NaN rows."""
    observations = []
    for step in [0, 1, 4, 5]:
        ts = Timestamp()
        ts.FromSeconds(360000 + (step * 3600))
        observations.append(TimeSeriesObservation(timestamp=ts, temperature_c=20.0 + step, humidity_pct=50.0))

    df = extract_observation_dataframe(observations, step=timedelta(hours=1))
    assert len(df) == 6
    assert df["temperature_c"].iloc[0] == 20.0
    assert df["temperature_c"].iloc[1] == 21.0
    assert np.isnan(df["temperature_c"].iloc[2])
    assert np.isnan(df["temperature_c"].iloc[3])
    assert df["temperature_c"].iloc[4] == 24.0
    assert df["temperature_c"].iloc[5] == 25.0


def test_extract_observation_dataframe_trailing_downtime_hourly_and_daily() -> None:
    """Test_extract_observation_dataframe_trailing_downtime_hourly_and_daily checks grid_end = issue_time - step."""
    observations = []
    for step in range(3):
        ts = Timestamp()
        ts.FromSeconds(360000 + (step * 3600))
        observations.append(TimeSeriesObservation(timestamp=ts, temperature_c=22.0, humidity_pct=55.0))

    issue_time = datetime.fromtimestamp(360000 + (5 * 3600), tz=UTC)
    df = extract_observation_dataframe(
        observations,
        step=timedelta(hours=1),
        issue_time=issue_time,
    )
    assert len(df) == 5
    assert not np.isnan(df["temperature_c"].iloc[2])
    assert np.isnan(df["temperature_c"].iloc[3])
    assert np.isnan(df["temperature_c"].iloc[4])

    # Daily: observations on days 0, 1, 2; issue_time on day 4 at 14:00 UTC -> grid_end is day 3 (00:00 UTC)
    daily_obs = []
    base_day = datetime(2026, 6, 1, 0, 0, tzinfo=UTC)
    for day in range(4):
        ts = Timestamp()
        ts.FromDatetime(base_day + timedelta(days=day))
        daily_obs.append(TimeSeriesObservation(timestamp=ts, temperature_c=20.0, humidity_pct=60.0))

    # Day 3 observation (2026-06-04 00:00) is the unfinished current day when issue_time is 2026-06-04 14:00
    daily_issue = datetime(2026, 6, 4, 14, 0, tzinfo=UTC)
    daily_df = extract_observation_dataframe(daily_obs, step=timedelta(days=1), issue_time=daily_issue)
    assert len(daily_df) == 3
    assert daily_df["timestamp"].iloc[-1] == datetime(2026, 6, 3, 0, 0, tzinfo=UTC)
