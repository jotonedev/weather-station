from typing import Any, cast

import numpy as np
import pytest

from inference.probability import exceedance_probability, quantile_at


def test_quantile_at_exact_and_interpolated() -> None:
    """Test_quantile_at_exact_and_interpolated verifies knot lookup and linear interpolation."""
    levels = (0.1, 0.5, 0.9)
    quantiles = np.array([[10.0, 20.0, 30.0]], dtype=np.float32)

    assert np.isclose(float(quantile_at(quantiles, levels, 0.1)[0]), 10.0)
    assert np.isclose(float(quantile_at(quantiles, levels, 0.5)[0]), 20.0)
    assert np.isclose(float(quantile_at(quantiles, levels, 0.9)[0]), 30.0)
    assert np.isclose(float(quantile_at(quantiles, levels, 0.3)[0]), 15.0)
    # Clamping beyond outer knots
    assert np.isclose(float(quantile_at(quantiles, levels, 0.01)[0]), 10.0)
    assert np.isclose(float(quantile_at(quantiles, levels, 0.99)[0]), 30.0)


def test_exceedance_probability_dry_partial_and_wet() -> None:
    """Test_exceedance_probability_dry_partial_and_wet verifies threshold crossing probabilities."""
    levels = (0.1, 0.5, 0.9)

    dry = np.array([0.0, 0.0, 0.0], dtype=np.float32)
    assert float(exceedance_probability(dry, levels, 0.2)) == 0.0

    # Zero median with historical upper-tail spike returns unscaled exceedance
    # x_left=0.0 (q=0.5), x_right=2.5 (q=0.9) -> cdf(0.2)=0.5 + (0.2/2.5)*0.4 = 0.532 -> 0.468
    background_tail = np.array([0.0, 0.0, 2.5], dtype=np.float32)
    assert np.isclose(float(exceedance_probability(background_tail, levels, 0.2)), 0.468)

    wet = np.array([1.0, 2.0, 5.0], dtype=np.float32)
    assert float(exceedance_probability(wet, levels, 0.2)) == 1.0

    # Transitional shower: q0.5=0.1, q0.9=0.3 -> cdf(0.2)=0.7 -> P(X>=0.2)=0.3
    tail_shower = np.array([0.0, 0.1, 0.3], dtype=np.float32)
    assert np.isclose(float(exceedance_probability(tail_shower, levels, 0.2)), 0.3)


def test_exceedance_probability_21_knots_zero_median() -> None:
    """Test_exceedance_probability_21_knots_zero_median verifies calibrated probability when p50 == 0.0."""
    levels = tuple(round(0.05 * idx, 2) for idx in range(21))
    curve = np.zeros(21, dtype=np.float32)
    # 0.0 through q=0.50 (index 10), then linearly rising to 1.0 at q=1.00 (index 20)
    # At q=0.60 (index 12), value is 0.20 -> P(X >= 0.20) = 0.40
    curve[10:] = np.linspace(0.0, 1.0, 11, dtype=np.float32)
    assert np.isclose(float(exceedance_probability(curve, levels, 0.2)), 0.40, atol=1e-5)


def test_probability_invalid_levels_raises() -> None:
    """Test_probability_invalid_levels_raises checks validation of mismatched level lengths."""
    quantiles = np.zeros((2, 3), dtype=np.float32)
    with pytest.raises(ValueError, match="does not match levels"):
        quantile_at(quantiles, (0.1, 0.5), 0.5)
    with pytest.raises(ValueError, match="non-empty"):
        exceedance_probability(quantiles, (), 0.2)
    with pytest.raises(ValueError, match="non-empty 1D"):
        quantile_at(quantiles, cast(Any, ((0.1, 0.5),)), 0.5)


def test_probability_crossing_quantiles_repaired() -> None:
    """Test_probability_crossing_quantiles_repaired repairs non-monotonic quantiles before evaluation."""
    levels = (0.1, 0.5, 0.9)
    crossing = np.array([[10.0, 25.0, 20.0]], dtype=np.float32)
    assert np.isclose(float(quantile_at(crossing, levels, 0.9)[0]), 25.0)
    assert np.isclose(float(quantile_at(crossing, levels, 0.7)[0]), 25.0)

    crossing_rain = np.array([0.0, 0.5, 0.1], dtype=np.float32)
    assert np.isclose(float(exceedance_probability(crossing_rain, levels, 0.2)), 0.74)


def test_exceedance_probability_edge_cases() -> None:
    """Test_exceedance_probability_edge_cases checks threshold boundaries, flat steps, and tensor shapes."""
    levels = (0.1, 0.5, 0.9)
    curve = np.array([1.0, 2.0, 3.0], dtype=np.float32)
    assert float(exceedance_probability(curve, levels, 0.0)) == 0.0
    assert float(exceedance_probability(curve, levels, -0.5)) == 0.0

    exact_min = np.array([0.2, 0.5, 1.0], dtype=np.float32)
    assert float(exceedance_probability(exact_min, levels, 0.2)) == 1.0

    flat_step = np.array([0.0, 0.2, 0.2, 1.0], dtype=np.float32)
    assert np.isclose(float(exceedance_probability(flat_step, (0.1, 0.4, 0.7, 0.9), 0.2)), 0.6)

    batch_quantiles = np.broadcast_to(np.array([0.0, 0.1, 0.3], dtype=np.float32), (2, 3, 3)).copy()
    batch_med = quantile_at(batch_quantiles, levels, 0.5)
    batch_prob = exceedance_probability(batch_quantiles, levels, 0.2)
    assert batch_med.shape == (2, 3)
    assert batch_prob.shape == (2, 3)
    assert np.allclose(batch_med, 0.1)
    assert np.allclose(batch_prob, 0.3)


def test_probability_single_knot_levels() -> None:
    """Test_probability_single_knot_levels handles single-level quantile tensors."""
    quantiles = np.array([[0.2], [0.1]], dtype=np.float32)
    levels = (0.5,)
    assert np.allclose(quantile_at(quantiles, levels, 0.1), [0.2, 0.1])
    assert np.allclose(quantile_at(quantiles, levels, 0.9), [0.2, 0.1])
    assert np.allclose(exceedance_probability(quantiles, levels, 0.2), [1.0, 0.0])
