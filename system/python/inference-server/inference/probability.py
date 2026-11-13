"""Quantile distribution utilities for point forecasts, intervals, and exceedance probabilities."""

from collections.abc import Sequence

import numpy as np

RAIN_THRESHOLD_HOURLY_MM = 0.2
RAIN_THRESHOLD_DAILY_MM = 1.0


def _validate_levels(quantiles: np.ndarray, levels: Sequence[float]) -> np.ndarray:
    """_validate_levels converts quantile levels to a validated 1D float64 array."""
    levels_arr = np.asarray(levels, dtype=np.float64)
    if levels_arr.ndim != 1 or levels_arr.size == 0:
        raise ValueError("quantile levels must be a non-empty 1D sequence")
    if quantiles.shape[-1] != levels_arr.size:
        raise ValueError(
            f"quantile tensor last dimension ({quantiles.shape[-1]}) does not match levels ({levels_arr.size})"
        )
    return levels_arr


def _interpolate_level(monotone: np.ndarray, levels_arr: np.ndarray, level: float) -> np.ndarray:
    """_interpolate_level linearly interpolates a monotone quantile tensor along its last axis."""
    clamped_level = float(np.clip(level, levels_arr[0], levels_arr[-1]))

    if levels_arr.size == 1:
        return monotone[..., 0]

    right_idx = int(np.searchsorted(levels_arr, clamped_level, side="left"))
    if right_idx <= 0:
        return monotone[..., 0]
    if right_idx >= levels_arr.size:
        return monotone[..., -1]

    left_idx = right_idx - 1
    q_left = levels_arr[left_idx]
    q_right = levels_arr[right_idx]
    if q_right <= q_left:
        return monotone[..., right_idx]

    weight = (clamped_level - q_left) / (q_right - q_left)
    return ((1.0 - weight) * monotone[..., left_idx]) + (weight * monotone[..., right_idx])


def quantile_at(quantiles: np.ndarray, levels: Sequence[float], level: float) -> np.ndarray:
    """quantile_at linearly interpolates the quantile tensor along its last axis at the requested level."""
    levels_arr = _validate_levels(quantiles, levels)
    monotone = np.maximum.accumulate(np.asarray(quantiles, dtype=np.float64), axis=-1)
    return _interpolate_level(monotone, levels_arr, level)


def _single_exceedance_probability(curve: np.ndarray, levels_arr: np.ndarray, threshold: float) -> float:
    """_single_exceedance_probability evaluates P(X >= threshold) from a monotone quantile curve."""
    if threshold <= 0.0 or curve[-1] < threshold:
        return 0.0
    if curve[0] >= threshold:
        return 1.0

    right_idx = int(np.searchsorted(curve, threshold, side="left"))
    left_idx = right_idx - 1
    x_left = float(curve[left_idx])
    x_right = float(curve[right_idx])
    q_left = float(levels_arr[left_idx])
    q_right = float(levels_arr[right_idx])

    if x_right <= x_left:
        cdf_at_threshold = q_right
    else:
        fraction = (threshold - x_left) / (x_right - x_left)
        cdf_at_threshold = q_left + (fraction * (q_right - q_left))
    return float(np.clip(1.0 - cdf_at_threshold, 0.0, 1.0))


def exceedance_probability(
    quantiles: np.ndarray,
    levels: Sequence[float],
    threshold: float,
) -> np.ndarray:
    """exceedance_probability computes P(X >= threshold) along the last axis of a quantile tensor."""
    levels_arr = _validate_levels(quantiles, levels)
    monotone = np.maximum.accumulate(np.asarray(quantiles, dtype=np.float64), axis=-1)

    flat = monotone.reshape(-1, levels_arr.size)
    probs = np.array(
        [_single_exceedance_probability(row, levels_arr, threshold) for row in flat],
        dtype=np.float64,
    )
    return probs.reshape(monotone.shape[:-1])
