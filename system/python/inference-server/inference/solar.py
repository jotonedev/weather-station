"""Astronomical and clear-sky solar geometry covariates for weather forecasting."""

from collections.abc import Sequence
from datetime import datetime, timedelta

import numpy as np

SOLAR_CONSTANT_WM2 = 1367.0
FAO56_SOLAR_CONSTANT_MJ_MIN = 0.0820


def _day_of_year_fraction(timestamps: Sequence[datetime], offset: timedelta) -> tuple[np.ndarray, np.ndarray]:
    """_day_of_year_fraction returns fractional day-of-year and UTC hour at bucket midpoints."""
    midpoints = [dt + offset for dt in timestamps]
    doy = np.array(
        [dt.timetuple().tm_yday + (dt.hour / 24.0) + (dt.minute / 1440.0) + (dt.second / 86400.0) for dt in midpoints],
        dtype=np.float64,
    )
    utc_hours = np.array(
        [dt.hour + (dt.minute / 60.0) + (dt.second / 3600.0) for dt in midpoints],
        dtype=np.float64,
    )
    return doy, utc_hours


def _solar_angles(
    doy: np.ndarray,
    latitude_deg: float,
) -> tuple[np.ndarray, np.ndarray, np.ndarray, np.ndarray, float]:
    """_solar_angles computes seasonal angle, inverse Earth-Sun distance, declination, and sunset hour angle."""
    doy_angle = (2.0 * np.pi * doy) / 365.25
    inv_dist_sq = 1.0 + (0.033 * np.cos(doy_angle))
    declination = 0.409 * np.sin(doy_angle - 1.39)
    lat_rad = float(np.radians(np.clip(latitude_deg, -89.5, 89.5)))
    cos_omega_s = np.clip(-np.tan(lat_rad) * np.tan(declination), -1.0, 1.0)
    omega_s = np.arccos(cos_omega_s)
    return doy_angle, inv_dist_sq, declination, omega_s, lat_rad


def _equation_of_time_hours(doy: np.ndarray) -> np.ndarray:
    """_equation_of_time_hours computes the Spencer equation-of-time correction in hours."""
    b_angle = (2.0 * np.pi * (doy - 81.0)) / 364.0
    eot_minutes = (9.87 * np.sin(2.0 * b_angle)) - (7.53 * np.cos(b_angle)) - (1.5 * np.sin(b_angle))
    return eot_minutes / 60.0


def hourly_known_covariates(
    timestamps: Sequence[datetime],
    latitude: float | None = None,
    longitude: float | None = None,
    elevation_m: float | None = None,
) -> dict[str, np.ndarray]:
    """hourly_known_covariates computes known solar and seasonal covariates at hourly bucket midpoints."""
    if not timestamps:
        return {}

    lat_deg = 0.0 if latitude is None or not np.isfinite(latitude) else float(latitude)
    lon_deg = 0.0 if longitude is None or not np.isfinite(longitude) else float(longitude)
    elev = 0.0 if elevation_m is None or not np.isfinite(elevation_m) else max(0.0, float(elevation_m))

    doy, utc_hours = _day_of_year_fraction(timestamps, timedelta(minutes=30))
    doy_angle, inv_dist_sq, declination, omega_s, lat_rad = _solar_angles(doy, lat_deg)

    eot_hours = _equation_of_time_hours(doy)
    solar_hours = np.mod(utc_hours + (lon_deg / 15.0) + eot_hours, 24.0)
    hour_angle = np.radians(15.0 * (solar_hours - 12.0))

    sin_elev = (np.sin(lat_rad) * np.sin(declination)) + (np.cos(lat_rad) * np.cos(declination) * np.cos(hour_angle))
    sin_elev = np.clip(sin_elev, -1.0, 1.0)

    clear_sky_transmittance = 0.75 + (2.0e-5 * elev)
    extraterrestrial_wm2 = SOLAR_CONSTANT_WM2 * inv_dist_sq * np.maximum(0.0, sin_elev)
    clear_sky_wm2 = clear_sky_transmittance * extraterrestrial_wm2

    solar_hour_angle = (2.0 * np.pi * solar_hours) / 24.0
    daylight_hours = (24.0 / np.pi) * omega_s

    return {
        "solar_elevation_sin": sin_elev.astype(np.float32),
        "clear_sky_irradiance_wm2": clear_sky_wm2.astype(np.float32),
        "solar_hour_sin": np.sin(solar_hour_angle).astype(np.float32),
        "solar_hour_cos": np.cos(solar_hour_angle).astype(np.float32),
        "doy_sin": np.sin(doy_angle).astype(np.float32),
        "doy_cos": np.cos(doy_angle).astype(np.float32),
        "daylight_hours": daylight_hours.astype(np.float32),
    }


def daily_known_covariates(
    timestamps: Sequence[datetime],
    latitude: float | None = None,
    elevation_m: float | None = None,
) -> dict[str, np.ndarray]:
    """daily_known_covariates computes daily clear-sky radiation, daylight duration, and seasonal angles."""
    if not timestamps:
        return {}

    lat_deg = 0.0 if latitude is None or not np.isfinite(latitude) else float(latitude)
    elev = 0.0 if elevation_m is None or not np.isfinite(elevation_m) else max(0.0, float(elevation_m))

    doy, _ = _day_of_year_fraction(timestamps, timedelta(hours=12))
    doy_angle, inv_dist_sq, declination, omega_s, lat_rad = _solar_angles(doy, lat_deg)

    # FAO-56 daily extraterrestrial radiation R_a in MJ m^-2 day^-1
    ra_mj = (
        ((24.0 * 60.0) / np.pi)
        * FAO56_SOLAR_CONSTANT_MJ_MIN
        * inv_dist_sq
        * (
            (omega_s * np.sin(lat_rad) * np.sin(declination))
            + (np.cos(lat_rad) * np.cos(declination) * np.sin(omega_s))
        )
    )
    ra_mj = np.maximum(0.0, ra_mj)
    clear_sky_mj = (0.75 + (2.0e-5 * elev)) * ra_mj
    daylight_hours = (24.0 / np.pi) * omega_s

    return {
        "clear_sky_radiation_mj": clear_sky_mj.astype(np.float32),
        "daylight_hours": daylight_hours.astype(np.float32),
        "doy_sin": np.sin(doy_angle).astype(np.float32),
        "doy_cos": np.cos(doy_angle).astype(np.float32),
    }
