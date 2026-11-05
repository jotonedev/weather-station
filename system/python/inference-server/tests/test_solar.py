import math
from datetime import UTC, datetime, timedelta

import numpy as np

from inference.solar import daily_known_covariates, hourly_known_covariates


def test_hourly_known_covariates_empty() -> None:
    """Test_hourly_known_covariates_empty returns an empty dictionary for no timestamps."""
    assert hourly_known_covariates([]) == {}


def test_daily_known_covariates_empty() -> None:
    """Test_daily_known_covariates_empty returns an empty dictionary for no timestamps."""
    assert daily_known_covariates([]) == {}


def test_hourly_known_covariates_shape_and_day_night_physics() -> None:
    """Test_hourly_known_covariates_shape_and_day_night_physics checks noon vs midnight irradiance."""
    start = datetime(2026, 6, 21, 0, 0, tzinfo=UTC)
    timestamps = [start + timedelta(hours=h) for h in range(24)]

    covs = hourly_known_covariates(
        timestamps,
        latitude=45.0,
        longitude=0.0,
        elevation_m=250.0,
    )

    expected_keys = {
        "solar_elevation_sin",
        "clear_sky_irradiance_wm2",
        "solar_hour_sin",
        "solar_hour_cos",
        "doy_sin",
        "doy_cos",
        "daylight_hours",
    }
    assert set(covs.keys()) == expected_keys
    for arr in covs.values():
        assert arr.shape == (24,)
        assert arr.dtype == np.float32
        assert np.isfinite(arr).all()

    # At 00:00 UTC (midpoint 00:30 UTC) at 0 deg longitude, the sun is below the horizon.
    assert covs["solar_elevation_sin"][0] < 0.0
    assert covs["clear_sky_irradiance_wm2"][0] == 0.0

    # At 12:00 UTC (midpoint 12:30 UTC), solar elevation and clear-sky irradiance are near peak.
    assert covs["solar_elevation_sin"][12] > 0.7
    assert covs["clear_sky_irradiance_wm2"][12] > 600.0
    assert covs["daylight_hours"][12] > 14.0


def test_hourly_known_covariates_longitude_shifts_solar_noon() -> None:
    """Test_hourly_known_covariates_longitude_shifts_solar_noon verifies local solar time shift."""
    ts = [datetime(2026, 3, 21, 18, 0, tzinfo=UTC)]

    # At 18:00 UTC, 0 deg longitude is evening, while -90 deg W longitude is local solar noon.
    greenwich = hourly_known_covariates(ts, latitude=30.0, longitude=0.0)
    west_90 = hourly_known_covariates(ts, latitude=30.0, longitude=-90.0)

    assert west_90["clear_sky_irradiance_wm2"][0] > greenwich["clear_sky_irradiance_wm2"][0]


def test_daily_known_covariates_summer_vs_winter() -> None:
    """Test_daily_known_covariates_summer_vs_winter verifies higher northern summer radiation."""
    dates = [
        datetime(2026, 6, 21, 0, 0, tzinfo=UTC),
        datetime(2026, 12, 21, 0, 0, tzinfo=UTC),
    ]
    covs = daily_known_covariates(dates, latitude=45.0, elevation_m=100.0)

    assert set(covs.keys()) == {
        "clear_sky_radiation_mj",
        "daylight_hours",
        "doy_sin",
        "doy_cos",
    }
    assert covs["clear_sky_radiation_mj"][0] > covs["clear_sky_radiation_mj"][1]
    assert covs["daylight_hours"][0] > covs["daylight_hours"][1]
    assert math.isclose(float(covs["daylight_hours"][0] + covs["daylight_hours"][1]), 24.0, abs_tol=0.5)
