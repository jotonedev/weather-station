from datetime import timedelta

import numpy as np
from google.protobuf.timestamp_pb2 import Timestamp
from weather.inference.v1.inference_pb2 import TimeSeriesObservation

from inference.covariates import prepare_context_and_covariates
from inference.features import extract_observation_dataframe


def test_prepare_hourly_targets_and_past_covariates() -> None:
    """Test_prepare_hourly_targets_and_past_covariates verifies target separation and derived signals."""
    observations = []
    for step in range(48):
        ts = Timestamp()
        ts.FromSeconds(1700000000 + (step * 3600))
        observations.append(
            TimeSeriesObservation(
                timestamp=ts,
                temperature_c=20.0 + (0.1 * step),
                min_temperature_c=18.0 + (0.1 * step),
                max_temperature_c=22.0 + (0.1 * step),
                humidity_pct=60.0,
                min_humidity_pct=55.0,
                max_humidity_pct=65.0,
                pressure_hpa=1015.0 - (0.1 * step),
                min_pressure_hpa=1014.0 - (0.1 * step),
                max_pressure_hpa=1016.0 - (0.1 * step),
                rainfall_mm=0.5,
                wind_speed_mps=4.0,
                max_wind_speed_mps=7.5,
                wind_u_mps=-2.0,
                wind_v_mps=3.5,
                max_rain_rate_mmh=2.5,
            )
        )

    df = extract_observation_dataframe(observations, step=timedelta(hours=1))
    prepared = prepare_context_and_covariates(df, is_daily=False)

    assert prepared.target_columns == ["temperature_c", "humidity_pct", "rainfall_mm", "pressure_hpa"]
    assert prepared.target.shape == (4, 48)
    assert "wind_gust_mps" in prepared.past_covariates
    assert "wind_u_mps" in prepared.past_covariates
    assert "wind_v_mps" in prepared.past_covariates
    assert "rain_rate_max_mmh" in prepared.past_covariates
    assert "dew_point_c" in prepared.past_covariates
    assert "dew_point_depression_c" in prepared.past_covariates
    assert "absolute_humidity_gm3" in prepared.past_covariates
    assert "pressure_tendency_3h_hpa" in prepared.past_covariates
    assert "pressure_tendency_24h_hpa" in prepared.past_covariates
    assert "rain_sum_24h_mm" in prepared.past_covariates

    assert np.isnan(prepared.past_covariates["pressure_tendency_3h_hpa"][:3]).all()
    assert np.isfinite(prepared.past_covariates["pressure_tendency_3h_hpa"][3:]).all()
    assert np.isnan(prepared.past_covariates["pressure_tendency_24h_hpa"][:24]).all()
    assert np.isfinite(prepared.past_covariates["pressure_tendency_24h_hpa"][24:]).all()
    assert np.isnan(prepared.past_covariates["rain_sum_24h_mm"][:23]).all()
    assert np.isclose(float(prepared.past_covariates["rain_sum_24h_mm"][23]), 12.0)


def test_prepare_daily_targets_and_past_covariates() -> None:
    """Test_prepare_daily_targets_and_past_covariates verifies daily targets and 7d rain totals."""
    observations = []
    for step in range(14):
        ts = Timestamp()
        ts.FromSeconds(1700000000 + (step * 86400))
        observations.append(
            TimeSeriesObservation(
                timestamp=ts,
                temperature_c=19.0,
                min_temperature_c=14.0,
                max_temperature_c=24.0,
                humidity_pct=65.0,
                min_humidity_pct=45.0,
                max_humidity_pct=85.0,
                pressure_hpa=1012.0,
                rainfall_mm=1.0,
            )
        )

    df = extract_observation_dataframe(observations, step=timedelta(days=1))
    prepared = prepare_context_and_covariates(df, is_daily=True)

    assert prepared.target_columns == [
        "min_temperature_c",
        "max_temperature_c",
        "min_humidity_pct",
        "max_humidity_pct",
        "rainfall_mm",
        "pressure_hpa",
    ]
    assert prepared.target.shape == (6, 14)
    assert "temperature_c" in prepared.past_covariates
    assert "humidity_pct" in prepared.past_covariates
    assert "rain_sum_7d_mm" in prepared.past_covariates
    assert "pressure_tendency_24h_hpa" in prepared.past_covariates
    assert np.isnan(prepared.past_covariates["pressure_tendency_24h_hpa"][0])
    assert np.isfinite(prepared.past_covariates["pressure_tendency_24h_hpa"][1:]).all()
    assert np.isnan(prepared.past_covariates["rain_sum_7d_mm"][:6]).all()
    assert np.isclose(float(prepared.past_covariates["rain_sum_7d_mm"][6]), 7.0)


def test_short_history_drops_incomplete_lag_covariates() -> None:
    """Test_short_history_drops_incomplete_lag_covariates ensures 24h derived covariates drop on 8h histories."""
    observations = []
    for step in range(8):
        ts = Timestamp()
        ts.FromSeconds(1700000000 + (step * 3600))
        observations.append(
            TimeSeriesObservation(
                timestamp=ts,
                temperature_c=20.0,
                humidity_pct=60.0,
                pressure_hpa=1013.0,
                rainfall_mm=0.5,
            )
        )

    df = extract_observation_dataframe(observations, step=timedelta(hours=1))
    prepared = prepare_context_and_covariates(df, is_daily=False)
    assert "pressure_tendency_3h_hpa" in prepared.past_covariates
    assert "pressure_tendency_24h_hpa" not in prepared.past_covariates
    assert "rain_sum_24h_mm" not in prepared.past_covariates


def test_short_gaps_interpolated_and_long_outages_kept_nan() -> None:
    """Test_short_gaps_interpolated_and_long_outages_kept_nan verifies 2-step bridge vs native NaN mask."""
    observations = []
    # Steps 0..19 with a 2-step interior gap at (2, 3) and a 5-step outage at (8..12)
    for step in range(20):
        if step in (2, 3) or (8 <= step <= 12):
            continue
        ts = Timestamp()
        ts.FromSeconds(1700000000 + (step * 3600))
        observations.append(
            TimeSeriesObservation(
                timestamp=ts,
                temperature_c=10.0 + step,
                humidity_pct=50.0,
            )
        )

    df = extract_observation_dataframe(observations, step=timedelta(hours=1))
    prepared = prepare_context_and_covariates(df, is_daily=False)
    temp_row = prepared.target[0]

    # 2-step interior gap at indices 2 and 3 is bridged linearly
    assert np.isclose(float(temp_row[2]), 12.0)
    assert np.isclose(float(temp_row[3]), 13.0)

    # 5-step outage leaves NaN beyond the 2-step limit for native model masking
    assert np.isnan(temp_row[10])
    assert np.isnan(temp_row[11])
    assert np.isnan(temp_row[12])
