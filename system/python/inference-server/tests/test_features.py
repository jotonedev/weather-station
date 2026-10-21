import math
from datetime import UTC, datetime

import numpy as np
from google.protobuf.timestamp_pb2 import Timestamp
from weather.inference.v1.inference_pb2 import TimeSeriesObservation

from inference.features import (
    MAX_GAP_HOURLY,
    calculate_temporal_covariates,
    extract_observation_dataframe,
    filter_and_impute_columns,
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


def test_wind_features_modeled() -> None:
    """Test_wind_features_modeled ensures wind speed, direction, and vector components enter the model."""
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
    _clean_df, surviving_columns = filter_and_impute_columns(dataframe)

    assert "wind_direction_deg" in surviving_columns
    assert "wind_speed_mps" in surviving_columns
    assert "wind_u" in surviving_columns
    assert "wind_v" in surviving_columns
    assert "temp_range_c" in surviving_columns
    assert "dew_point_c" in surviving_columns


def test_column_filtering_sparse_column_dropped() -> None:
    """Test_column_filtering_sparse_column_dropped verifies dropping columns with missing ratio over 30 percent."""
    observations = []
    for step_index in range(10):
        timestamp = Timestamp()
        timestamp.FromSeconds(1000 + (step_index * 3600))
        observation = TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=20.0,
            humidity_pct=50.0,
        )
        # Supply rainfall only for 6 out of 10 observations (40 percent missing)
        if step_index < 6:
            observation.rainfall_mm = 1.2

        observations.append(observation)

    dataframe = extract_observation_dataframe(observations)
    _clean_df, surviving_columns = filter_and_impute_columns(dataframe)

    assert "rainfall_mm" not in surviving_columns
    assert "temperature_c" in surviving_columns
    assert "humidity_pct" in surviving_columns


def test_column_filtering_dense_column_retained() -> None:
    """Test_column_filtering_dense_column_retained verifies keeping columns with missing ratio under 30 percent."""
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
        # Supply rainfall for 8 out of 10 observations (20 percent missing)
        if step_index < 8:
            observation.rainfall_mm = 0.5

        observations.append(observation)

    dataframe = extract_observation_dataframe(observations)
    clean_df, surviving_columns = filter_and_impute_columns(dataframe)

    assert "rainfall_mm" in surviving_columns
    assert "pressure_hpa" in surviving_columns
    assert not clean_df.isna().any().any()


def test_pressure_sparse_column_dropped() -> None:
    """Test_pressure_sparse_column_dropped verifies pressure is excluded above the 30 percent missing ratio."""
    observations = []
    for step_index in range(10):
        timestamp = Timestamp()
        timestamp.FromSeconds(1000 + (step_index * 3600))
        observation = TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=20.0,
            humidity_pct=50.0,
        )
        if step_index < 6:
            observation.pressure_hpa = 1013.0

        observations.append(observation)

    dataframe = extract_observation_dataframe(observations)
    _clean_df, surviving_columns = filter_and_impute_columns(dataframe)

    assert "pressure_hpa" not in surviving_columns


def test_large_gap_not_fully_interpolated() -> None:
    """Test_large_gap_not_fully_interpolated ensures oversized gaps never become one straight line."""
    observations = []
    gap_start = 5
    # Gap far beyond the bridging cap (2 * MAX_GAP_HOURLY + 20).
    gap_length = (MAX_GAP_HOURLY * 2) + 20
    total = gap_start + gap_length + 5
    for step_index in range(total):
        timestamp = Timestamp()
        timestamp.FromSeconds(1000 + (step_index * 3600))
        if gap_start <= step_index < gap_start + gap_length:
            temperature = float("nan")
        else:
            temperature = 10.0 if step_index < gap_start else 30.0
        observation = TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=temperature,
            humidity_pct=60.0,
        )
        observations.append(observation)

    dataframe = extract_observation_dataframe(observations)
    clean_df, _surviving_columns = filter_and_impute_columns(dataframe)

    values = clean_df["temperature_c"].to_numpy()
    gap_values = values[gap_start : gap_start + gap_length]
    # A fully interpolated gap would be strictly monotonic (one ramp). The cap
    # must leave at least one constant plateau inside the gap.
    assert np.any(np.diff(gap_values) == 0.0)
    assert values[gap_start - 1] == 10.0
    assert values[gap_start + gap_length] == 30.0
    # The edges of the gap must not be bridged all the way to the other side.
    assert gap_values[0] < 30.0
    assert np.isfinite(values).all()


def test_inf_normalized_to_missing_and_imputed() -> None:
    """Test_inf_normalized_to_missing_and_imputed ensures infinities never reach the model input."""
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
    # Extraction normalizes infinities into missing values.
    assert dataframe["temperature_c"].iloc[2] != float("inf")
    assert dataframe["temperature_c"].isna().sum() == 2
    assert dataframe["temperature_c"].isna().sum() <= 3

    clean_df, _surviving_columns = filter_and_impute_columns(dataframe)
    values = clean_df["temperature_c"].to_numpy()
    assert np.isfinite(values).all()


def test_calculate_temporal_covariates() -> None:
    """Test_calculate_temporal_covariates verifies sinusoidal bounds and shape."""
    timestamps = [
        datetime(2026, 6, 21, 0, 0, tzinfo=UTC),
        datetime(2026, 6, 21, 6, 0, tzinfo=UTC),
        datetime(2026, 6, 21, 12, 0, tzinfo=UTC),
    ]

    covariates = calculate_temporal_covariates(timestamps)
    assert "hour_sin" in covariates
    assert "hour_cos" in covariates
    assert "doy_sin" in covariates
    assert "doy_cos" in covariates
    assert len(covariates["hour_sin"]) == 3

    # At 00:00 UTC, hour_sin is 0 and hour_cos is 1
    assert math.isclose(float(covariates["hour_sin"][0]), 0.0, abs_tol=1e-5)
    assert math.isclose(float(covariates["hour_cos"][0]), 1.0, abs_tol=1e-5)


def test_generate_future_timestamps() -> None:
    """Test_generate_future_timestamps checks sequence lengths for hourly and daily modes."""
    issue_time = datetime(2026, 1, 1, 12, 0, tzinfo=UTC)

    hourly_steps = generate_hourly_future_timestamps(issue_time, 24)
    assert len(hourly_steps) == 24
    assert hourly_steps[0] == datetime(2026, 1, 1, 13, 0, tzinfo=UTC)
    assert hourly_steps[-1] == datetime(2026, 1, 2, 12, 0, tzinfo=UTC)

    daily_steps = generate_daily_future_timestamps(issue_time, 10)
    assert len(daily_steps) == 10
    assert daily_steps[0] == datetime(2026, 1, 2, 12, 0, tzinfo=UTC)
    assert daily_steps[-1] == datetime(2026, 1, 11, 12, 0, tzinfo=UTC)
