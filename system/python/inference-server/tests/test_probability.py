import numpy as np
import pytest

from inference.probability import exceedance_probability, quantile_at, quantile_mean


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


def test_quantile_mean_symmetric_and_skewed() -> None:
    """Test_quantile_mean_symmetric_and_skewed checks trapezoidal integration across knots."""
    levels = (0.1, 0.5, 0.9)
    symmetric = np.array([10.0, 20.0, 30.0], dtype=np.float32)
    assert np.isclose(float(quantile_mean(symmetric, levels)), 20.0)

    # Skewed rain distribution where median is 0.0 but upper tail has rain
    skewed_rain = np.array([0.0, 0.0, 4.0], dtype=np.float32)
    expected = float(quantile_mean(skewed_rain, levels, clip_min=0.0))
    assert expected > 0.0
    assert np.isclose(expected, 1.2)


def test_exceedance_probability_dry_partial_and_wet() -> None:
    """Test_exceedance_probability_dry_partial_and_wet verifies threshold crossing probabilities."""
    levels = (0.1, 0.5, 0.9)

    dry = np.array([0.0, 0.0, 0.0], dtype=np.float32)
    assert float(exceedance_probability(dry, levels, 0.2)) == 0.0

    wet = np.array([1.0, 2.0, 5.0], dtype=np.float32)
    assert float(exceedance_probability(wet, levels, 0.2)) == 1.0

    # Threshold 0.2 is halfway between q0.5=0.0 and q0.9=0.4 -> CDF=0.7 -> P(X>=0.2)=0.3
    tail_shower = np.array([0.0, 0.0, 0.4], dtype=np.float32)
    assert np.isclose(float(exceedance_probability(tail_shower, levels, 0.2)), 0.3)


def test_probability_invalid_levels_raises() -> None:
    """Test_probability_invalid_levels_raises checks validation of mismatched level lengths."""
    quantiles = np.zeros((2, 3), dtype=np.float32)
    with pytest.raises(ValueError, match="does not match levels"):
        quantile_at(quantiles, (0.1, 0.5), 0.5)
    with pytest.raises(ValueError, match="non-empty"):
        quantile_mean(quantiles, (), clip_min=0.0)
