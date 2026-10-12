import math
from datetime import UTC, datetime

from google.protobuf.timestamp_pb2 import Timestamp
from weather.inference.v1.inference_pb2 import (
    PREDICTION_MODE_NEXT_10_DAYS,
    PREDICTION_MODE_NEXT_24_HOURS,
    TimeSeriesObservation,
)

from inference.features import (
    calculate_temporal_covariates,
    extract_observation_dataframe,
    filter_and_impute_columns,
    generate_future_timestamps,
    reconstruct_wind_direction,
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


def test_wind_decomposition_and_reconstruction() -> None:
    """Test_wind_decomposition_and_reconstruction verifies vector projection and angle recovery."""
    angles_to_test = [0.0, 45.0, 90.0, 180.0, 270.0, 359.0]
    for angle in angles_to_test:
        rad = math.radians(angle)
        vector_u = math.sin(rad)
        vector_v = math.cos(rad)
        recovered_angle = reconstruct_wind_direction(vector_u, vector_v)
        assert math.isclose(angle, recovered_angle, abs_tol=1e-4)


def test_wind_ignored_when_partially_null() -> None:
    """Test_wind_ignored_when_partially_null ensures wind is excluded when speed or direction is missing."""
    timestamp = Timestamp()
    timestamp.FromSeconds(1000)

    observations = [
        TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=20.0,
            humidity_pct=50.0,
            wind_speed_mps=5.0,
        ),
        TimeSeriesObservation(
            timestamp=timestamp,
            temperature_c=21.0,
            humidity_pct=52.0,
            wind_direction_deg=90.0,
        ),
    ]

    dataframe = extract_observation_dataframe(observations)
    _clean_df, surviving_columns = filter_and_impute_columns(dataframe)

    assert "wind_u" not in surviving_columns
    assert "wind_v" not in surviving_columns


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
            wind_speed_mps=4.0,
            wind_direction_deg=180.0,
        )
        # Supply rainfall for 8 out of 10 observations (20 percent missing)
        if step_index < 8:
            observation.rainfall_mm = 0.5

        observations.append(observation)

    dataframe = extract_observation_dataframe(observations)
    clean_df, surviving_columns = filter_and_impute_columns(dataframe)

    assert "rainfall_mm" in surviving_columns
    assert "wind_u" in surviving_columns
    assert "wind_v" in surviving_columns
    assert not clean_df.isna().any().any()


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

    hourly_steps = generate_future_timestamps(issue_time, PREDICTION_MODE_NEXT_24_HOURS)
    assert len(hourly_steps) == 24
    assert hourly_steps[0] == datetime(2026, 1, 1, 13, 0, tzinfo=UTC)
    assert hourly_steps[-1] == datetime(2026, 1, 2, 12, 0, tzinfo=UTC)

    daily_steps = generate_future_timestamps(issue_time, PREDICTION_MODE_NEXT_10_DAYS)
    assert len(daily_steps) == 10
    assert daily_steps[0] == datetime(2026, 1, 2, 12, 0, tzinfo=UTC)
    assert daily_steps[-1] == datetime(2026, 1, 11, 12, 0, tzinfo=UTC)
