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

    # Zero median with historical upper-tail spike is suppressed as dry background noise
    background_tail = np.array([0.0, 0.0, 2.5], dtype=np.float32)
    assert float(exceedance_probability(background_tail, levels, 0.2)) == 0.0

    wet = np.array([1.0, 2.0, 5.0], dtype=np.float32)
    assert float(exceedance_probability(wet, levels, 0.2)) == 1.0

    # Transitional shower: q0.5=0.1, q0.9=0.3 -> raw P(X>=0.2)=0.3, scaled by p50/threshold=0.5 -> 0.15
    tail_shower = np.array([0.0, 0.1, 0.3], dtype=np.float32)
    assert np.isclose(float(exceedance_probability(tail_shower, levels, 0.2)), 0.15)


def test_probability_invalid_levels_raises() -> None:
    """Test_probability_invalid_levels_raises checks validation of mismatched level lengths."""
    quantiles = np.zeros((2, 3), dtype=np.float32)
    with pytest.raises(ValueError, match="does not match levels"):
        quantile_at(quantiles, (0.1, 0.5), 0.5)
    with pytest.raises(ValueError, match="non-empty"):
        exceedance_probability(quantiles, (), 0.2)
