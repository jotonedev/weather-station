import math
from datetime import UTC, datetime, timedelta

import numpy as np

from inference.solar import daily_known_covariates, hourly_known_covariates


def test_known_covariates_empty() -> None:
    """Test_known_covariates_empty returns empty dictionaries for no timestamps."""
    assert hourly_known_covariates([]) == {}
    assert daily_known_covariates([]) == {}


def test_hourly_known_covariates_shape_and_day_night_physics() -> None:
    """Test_hourly_known_covariates_shape_and_day_night_physics checks noon vs midnight irradiance."""
    start = datetime(2026, 6, 21, 0, 0, tzinfo=UTC)
    timestamps = [start + timedelta(hours=hour_offset) for hour_offset in range(24)]

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


def test_hourly_known_covariates_3h_midpoint() -> None:
    """Test_hourly_known_covariates_3h_midpoint uses bucket midpoint (+90m) for 3h steps."""
    bucket_start = [datetime(2026, 6, 21, 9, 0, tzinfo=UTC)]
    covs_3h = hourly_known_covariates(bucket_start, latitude=45.0, longitude=0.0, step=timedelta(hours=3))
    covs_1h_mid = hourly_known_covariates(
        [datetime(2026, 6, 21, 10, 0, tzinfo=UTC)],
        latitude=45.0,
        longitude=0.0,
        step=timedelta(hours=1),
    )
    assert np.isclose(float(covs_3h["solar_elevation_sin"][0]), float(covs_1h_mid["solar_elevation_sin"][0]))


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


def test_known_covariates_non_finite_and_negative_elevation_fallback() -> None:
    """Test_known_covariates_non_finite_and_negative_elevation_fallback clamps invalid location inputs."""
    timestamps = [datetime(2026, 6, 21, 12, 0, tzinfo=UTC)]

    hourly_baseline = hourly_known_covariates(timestamps, latitude=0.0, longitude=0.0, elevation_m=0.0)
    hourly_non_finite = hourly_known_covariates(
        timestamps,
        latitude=float("nan"),
        longitude=float("inf"),
        elevation_m=float("-inf"),
    )
    hourly_neg_elev = hourly_known_covariates(timestamps, latitude=0.0, longitude=0.0, elevation_m=-500.0)

    for key, expected in hourly_baseline.items():
        assert np.isfinite(hourly_non_finite[key]).all()
        assert hourly_non_finite[key].dtype == np.float32
        np.testing.assert_allclose(hourly_non_finite[key], expected)
        np.testing.assert_allclose(hourly_neg_elev[key], expected)

    daily_baseline = daily_known_covariates(timestamps, latitude=0.0, elevation_m=0.0)
    daily_non_finite = daily_known_covariates(timestamps, latitude=float("nan"), elevation_m=float("inf"))
    daily_neg_elev = daily_known_covariates(timestamps, latitude=0.0, elevation_m=-500.0)

    for key, expected in daily_baseline.items():
        assert np.isfinite(daily_non_finite[key]).all()
        assert daily_non_finite[key].dtype == np.float32
        np.testing.assert_allclose(daily_non_finite[key], expected)
        np.testing.assert_allclose(daily_neg_elev[key], expected)


def test_known_covariates_polar_latitude_clamping() -> None:
    """Test_known_covariates_polar_latitude_clamping avoids tangent blowup at +/-90 degrees."""
    solstice = [datetime(2026, 6, 21, 12, 0, tzinfo=UTC)]

    hourly_north_ref = hourly_known_covariates(solstice, latitude=89.5, longitude=0.0, elevation_m=0.0)
    hourly_south_ref = hourly_known_covariates(solstice, latitude=-89.5, longitude=0.0, elevation_m=0.0)
    for lat_north in (90.0, 100.0):
        north = hourly_known_covariates(solstice, latitude=lat_north, longitude=0.0, elevation_m=0.0)
        for key, expected in hourly_north_ref.items():
            assert np.isfinite(north[key]).all()
            np.testing.assert_allclose(north[key], expected)
    for lat_south in (-90.0, -100.0):
        south = hourly_known_covariates(solstice, latitude=lat_south, longitude=0.0, elevation_m=0.0)
        for key, expected in hourly_south_ref.items():
            assert np.isfinite(south[key]).all()
            np.testing.assert_allclose(south[key], expected)

    daily_north_ref = daily_known_covariates(solstice, latitude=89.5, elevation_m=0.0)
    daily_south_ref = daily_known_covariates(solstice, latitude=-89.5, elevation_m=0.0)
    for lat_north in (90.0, 100.0):
        north_daily = daily_known_covariates(solstice, latitude=lat_north, elevation_m=0.0)
        for key, expected in daily_north_ref.items():
            assert np.isfinite(north_daily[key]).all()
            np.testing.assert_allclose(north_daily[key], expected)
        assert np.isclose(float(north_daily["daylight_hours"][0]), 24.0)
    for lat_south in (-90.0, -100.0):
        south_daily = daily_known_covariates(solstice, latitude=lat_south, elevation_m=0.0)
        for key, expected in daily_south_ref.items():
            assert np.isfinite(south_daily[key]).all()
            np.testing.assert_allclose(south_daily[key], expected)
        assert float(south_daily["daylight_hours"][0]) == 0.0
        assert float(south_daily["clear_sky_radiation_mj"][0]) == 0.0
