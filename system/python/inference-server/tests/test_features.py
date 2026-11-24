from datetime import UTC, datetime, timedelta

import numpy as np
import pytest
from google.protobuf.timestamp_pb2 import Timestamp
from weather.inference.v1.inference_pb2 import TimeSeriesObservation

from inference.covariates import prepare_context_and_covariates
from inference.features import (
    DAILY_BUCKET_STEP,
    compute_daily_substep_plan,
    extract_observation_dataframe,
    floor_to_day,
    floor_to_hour,
    floor_to_step,
    generate_daily_future_timestamps,
    generate_hourly_future_timestamps,
    generate_step_future_timestamps,
    sort_observations,
)


def test_sort_observations() -> None:
    """Test_sort_observations verifies observations sort ascending by (seconds, nanos)."""
    timestamp_early = Timestamp(seconds=100, nanos=100_000_000)
    timestamp_late_same_sec = Timestamp(seconds=100, nanos=500_000_000)
    timestamp_next_sec = Timestamp(seconds=200, nanos=0)

    observation_next = TimeSeriesObservation(timestamp=timestamp_next_sec)
    observation_late_same_sec = TimeSeriesObservation(timestamp=timestamp_late_same_sec)
    observation_early = TimeSeriesObservation(timestamp=timestamp_early)

    sorted_result = sort_observations([observation_next, observation_late_same_sec, observation_early])
    assert [(obs.timestamp.seconds, obs.timestamp.nanos) for obs in sorted_result] == [
        (100, 100_000_000),
        (100, 500_000_000),
        (200, 0),
    ]


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


@pytest.mark.parametrize(
    ("present_count", "expected_retained"),
    [
        (4, False),
        (5, True),
        (6, True),
    ],
)
def test_column_filtering_density_threshold(present_count: int, expected_retained: bool) -> None:
    """Test_column_filtering_density_threshold keeps columns up to 50% missing and drops sparser ones."""
    observations = []
    for step_index in range(10):
        timestamp = Timestamp()
        timestamp.FromSeconds(1000 + (step_index * 3600))
        observation = TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=20.0,
            humidity_pct=50.0,
        )
        if step_index < present_count:
            observation.rainfall_mm = 0.5
            observation.pressure_hpa = 1013.0

        observations.append(observation)

    dataframe = extract_observation_dataframe(observations)
    prepared = prepare_context_and_covariates(dataframe)

    assert ("rainfall_mm" in prepared.target_columns) is expected_retained
    assert ("pressure_hpa" in prepared.target_columns) is expected_retained
    assert "temperature_c" in prepared.target_columns
    assert "humidity_pct" in prepared.target_columns


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
    """Test_generate_future_timestamps_contiguous_with_past checks flooring and bucket starts."""
    issue_time = datetime(2026, 1, 1, 12, 35, 22, tzinfo=UTC)

    assert floor_to_hour(issue_time) == datetime(2026, 1, 1, 12, 0, 0, tzinfo=UTC)
    assert floor_to_day(issue_time) == datetime(2026, 1, 1, 0, 0, 0, tzinfo=UTC)

    hourly_steps = generate_hourly_future_timestamps(issue_time, 24)
    assert len(hourly_steps) == 24
    assert hourly_steps[0] == datetime(2026, 1, 1, 12, 0, tzinfo=UTC)
    assert hourly_steps[-1] == datetime(2026, 1, 2, 11, 0, tzinfo=UTC)

    daily_steps = generate_daily_future_timestamps(issue_time, 10)
    assert len(daily_steps) == 10
    assert daily_steps[0] == datetime(2026, 1, 1, 0, 0, tzinfo=UTC)
    assert daily_steps[-1] == datetime(2026, 1, 10, 0, 0, tzinfo=UTC)

    assert floor_to_step(datetime(2026, 1, 1, 8, 45, tzinfo=UTC), DAILY_BUCKET_STEP) == datetime(
        2026, 1, 1, 6, 0, tzinfo=UTC
    )
    step_future = generate_step_future_timestamps(datetime(2026, 1, 1, 8, 45, tzinfo=UTC), 3, DAILY_BUCKET_STEP)
    assert step_future == [
        datetime(2026, 1, 1, 6, 0, tzinfo=UTC),
        datetime(2026, 1, 1, 9, 0, tzinfo=UTC),
        datetime(2026, 1, 1, 12, 0, tzinfo=UTC),
    ]


@pytest.mark.parametrize(
    ("hour_utc", "expected_grid_hour", "expected_len", "expected_first_eod", "expected_last_eod"),
    [
        (0, 0, 80, 7, 79),
        (8, 6, 78, 5, 77),
        (22, 21, 73, 0, 72),
    ],
)
def test_compute_daily_substep_plan(
    hour_utc: int,
    expected_grid_hour: int,
    expected_len: int,
    expected_first_eod: int,
    expected_last_eod: int,
) -> None:
    """Test_compute_daily_substep_plan aligns 3h grid issue time and end-of-day indices."""
    issue_time = datetime(2026, 11, 24, hour_utc, 15, tzinfo=UTC)
    grid_issue, pred_len, eod_indices = compute_daily_substep_plan(issue_time, horizon_days=10, step=DAILY_BUCKET_STEP)

    assert grid_issue == datetime(2026, 11, 24, expected_grid_hour, 0, tzinfo=UTC)
    assert pred_len == expected_len
    assert len(eod_indices) == 10
    assert eod_indices[0] == expected_first_eod
    assert eod_indices[-1] == expected_last_eod


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


def test_missing_wind_speed_leaves_wind_components_nan() -> None:
    """Test_missing_wind_speed_leaves_wind_components_nan verifies no synthetic 1.0 m/s fallback is used."""
    ts = Timestamp()
    ts.FromSeconds(360000)
    obs = [
        TimeSeriesObservation(
            timestamp=ts,
            temperature_c=20.0,
            humidity_pct=50.0,
            wind_direction_deg=90.0,
        )
    ]
    df = extract_observation_dataframe(obs)
    assert np.isnan(df["wind_u_mps"].iloc[0])
    assert np.isnan(df["wind_v_mps"].iloc[0])


def test_missing_daily_min_max_targets_remain_nan() -> None:
    """Test_missing_daily_min_max_targets_remain_nan verifies absent min/max fields are not copied from mean."""
    ts = Timestamp()
    ts.FromSeconds(360000)
    obs = [
        TimeSeriesObservation(
            timestamp=ts,
            temperature_c=20.0,
            humidity_pct=50.0,
            pressure_hpa=1013.0,
            wind_speed_mps=4.0,
        )
    ]
    df = extract_observation_dataframe(obs)
    for col in (
        "min_temperature_c",
        "max_temperature_c",
        "min_humidity_pct",
        "max_humidity_pct",
        "min_pressure_hpa",
        "max_pressure_hpa",
        "wind_gust_mps",
    ):
        assert np.isnan(df[col].iloc[0])


def test_extract_observation_dataframe_empty_when_no_completed_observations() -> None:
    """Test_extract_observation_dataframe_empty_when_no_completed_observations returns empty DataFrame."""
    assert extract_observation_dataframe([]).empty

    issue_time = datetime(2026, 6, 1, 12, 0, tzinfo=UTC)
    ts = Timestamp()
    ts.FromDatetime(issue_time)
    df = extract_observation_dataframe(
        [TimeSeriesObservation(timestamp=ts, temperature_c=20.0, humidity_pct=50.0)],
        step=timedelta(hours=1),
        issue_time=issue_time,
    )
    assert df.empty


def test_wind_features_partial_components() -> None:
    """Test_wind_features_partial_components preserves provided wind components and derives missing ones."""
    ts = Timestamp()
    ts.FromSeconds(360000)

    u_with_speed_df = extract_observation_dataframe(
        [
            TimeSeriesObservation(
                timestamp=ts,
                temperature_c=20.0,
                humidity_pct=50.0,
                wind_u_mps=1.5,
                wind_speed_mps=5.0,
                wind_direction_deg=180.0,
            )
        ]
    )
    assert np.isclose(float(u_with_speed_df["wind_u_mps"].iloc[0]), 1.5)
    assert np.isclose(float(u_with_speed_df["wind_v_mps"].iloc[0]), 5.0)

    v_with_speed_df = extract_observation_dataframe(
        [
            TimeSeriesObservation(
                timestamp=ts,
                temperature_c=20.0,
                humidity_pct=50.0,
                wind_v_mps=-2.0,
                wind_speed_mps=5.0,
                wind_direction_deg=90.0,
            )
        ]
    )
    assert np.isclose(float(v_with_speed_df["wind_u_mps"].iloc[0]), -5.0)
    assert np.isclose(float(v_with_speed_df["wind_v_mps"].iloc[0]), -2.0)

    u_only_df = extract_observation_dataframe(
        [TimeSeriesObservation(timestamp=ts, temperature_c=20.0, humidity_pct=50.0, wind_u_mps=1.5)]
    )
    assert np.isclose(float(u_only_df["wind_u_mps"].iloc[0]), 1.5)
    assert np.isnan(u_only_df["wind_v_mps"].iloc[0])

    v_only_df = extract_observation_dataframe(
        [TimeSeriesObservation(timestamp=ts, temperature_c=20.0, humidity_pct=50.0, wind_v_mps=-2.0)]
    )
    assert np.isnan(v_only_df["wind_u_mps"].iloc[0])
    assert np.isclose(float(v_only_df["wind_v_mps"].iloc[0]), -2.0)
