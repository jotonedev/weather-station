from typing import cast

import numpy as np
import pytest

pytest.importorskip("timestral")

import torch
from timestral import Forecast, TimestralForecaster

from inference.backends.timestral import TimestralBackend, _level_index, _stack_known_covariates
from tests.helpers import create_synthetic_timestral_forecaster

LEVELS = (0.05, 0.1, 0.25, 0.4, 0.5, 0.6, 0.75, 0.9, 0.95)


def test_level_index_exact_match() -> None:
    """Test_level_index_exact_match verifies requested levels resolve to their knot position."""
    assert _level_index(LEVELS, 0.1) == 1
    assert _level_index(LEVELS, 0.5) == 4
    assert _level_index(LEVELS, 0.9) == 7


def test_level_index_nearest_within_tolerance() -> None:
    """Test_level_index_nearest_within_tolerance verifies tiny drift snaps to the closest knot."""
    assert _level_index(LEVELS, 0.1001) == 1


def test_level_index_missing_level_raises() -> None:
    """Test_level_index_missing_level_raises verifies a level far from all knots is rejected."""
    with pytest.raises(ValueError, match="does not provide quantile level"):
        _level_index(LEVELS, 0.45)


def test_stack_known_covariates_concatenates_past_and_future() -> None:
    """Test_stack_known_covariates_concatenates_past_and_future verifies the known-future tensor layout."""
    past = {"hour_sin": np.array([1.0, 2.0]), "hour_cos": np.array([3.0, 4.0])}
    future = {"hour_sin": np.array([5.0]), "hour_cos": np.array([6.0])}

    stacked = _stack_known_covariates(past, future)

    assert stacked is not None
    assert stacked.shape == (1, 2, 3)
    assert stacked.dtype == np.float32
    np.testing.assert_array_equal(stacked[0, 0], [1.0, 2.0, 5.0])
    np.testing.assert_array_equal(stacked[0, 1], [3.0, 4.0, 6.0])


def test_stack_known_covariates_empty_returns_none() -> None:
    """Test_stack_known_covariates_empty_returns_none verifies no covariates yields None."""
    assert _stack_known_covariates({}, {}) is None


def test_stack_known_covariates_missing_future_keys_raises() -> None:
    """Test_stack_known_covariates_missing_future_keys_raises verifies mismatched covariate keys are rejected."""
    past = {"hour_sin": np.array([1.0])}
    with pytest.raises(ValueError, match="missing keys"):
        _stack_known_covariates(past, {})


def test_timestral_max_recommended_horizon_unbounded() -> None:
    """Test_timestral_max_recommended_horizon_unbounded reports no prediction length cap."""
    forecaster = create_synthetic_timestral_forecaster()
    backend = TimestralBackend(model_name="synthetic-test", pipeline=forecaster)

    assert backend.max_recommended_horizon() is None


def test_timestral_predict_with_synthetic_model() -> None:
    """Test_timestral_predict_with_synthetic_model verifies a full CPU forecast through the backend."""
    forecaster = create_synthetic_timestral_forecaster()
    backend = TimestralBackend(model_name="synthetic-test", pipeline=forecaster)

    history_length = 40
    prediction_length = 16
    target = np.random.randn(3, history_length).astype(np.float32)
    covariate_names = ("hour_sin", "hour_cos", "doy_sin", "doy_cos")
    past_covariates = {name: np.random.randn(history_length).astype(np.float32) for name in covariate_names}
    future_covariates = {name: np.random.randn(prediction_length).astype(np.float32) for name in covariate_names}

    quantiles, mean = backend.predict(
        target=target,
        past_covariates=past_covariates,
        future_covariates=future_covariates,
        prediction_length=prediction_length,
    )

    assert quantiles.shape == (3, prediction_length, 3)
    assert mean.shape == (3, prediction_length)
    assert np.isfinite(quantiles).all()
    assert np.isfinite(mean).all()
    # Quantile columns must stay ordered 0.1 <= 0.5 <= 0.9
    assert (quantiles[..., 0] <= quantiles[..., 1] + 1e-5).all()
    assert (quantiles[..., 1] <= quantiles[..., 2] + 1e-5).all()


class RecordingForecaster:
    """RecordingForecaster records forecast arguments and returns a fixed Forecast object."""

    def __init__(self):
        self.captured_context = None
        self.captured_prediction_length = None
        self.captured_known_covariates: np.ndarray | None = None
        self.quantile_levels = (0.1, 0.3, 0.5, 0.7, 0.9)

    def forecast(self, context, prediction_length, known_covariates=None):
        """forecast captures arguments and returns quantiles holding the level index at every position."""
        self.captured_context = context
        self.captured_prediction_length = prediction_length
        self.captured_known_covariates = known_covariates
        quantiles = torch.arange(5, dtype=torch.float32).reshape(1, 1, 1, 5).expand(1, 2, 8, 5).clone()
        return Forecast(quantiles=quantiles, quantile_levels=self.quantile_levels)


def test_timestral_predict_maps_inputs_and_levels() -> None:
    """Test_timestral_predict_maps_inputs_and_levels verifies context, covariates, and level extraction."""
    forecaster = RecordingForecaster()
    backend = TimestralBackend(model_name="synthetic-test", pipeline=cast(TimestralForecaster, forecaster))

    target = np.zeros((2, 10), dtype=np.float32)
    past_covariates = {"hour_sin": np.ones(10, dtype=np.float32)}
    future_covariates = {"hour_sin": 2.0 * np.ones(8, dtype=np.float32)}

    quantiles, mean = backend.predict(
        target=target,
        past_covariates=past_covariates,
        future_covariates=future_covariates,
        prediction_length=8,
    )

    assert forecaster.captured_prediction_length == 8
    np.testing.assert_array_equal(forecaster.captured_context, target[None, ...])
    known_covariates = forecaster.captured_known_covariates
    assert known_covariates is not None
    assert known_covariates.shape == (1, 1, 18)
    np.testing.assert_array_equal(
        known_covariates[0, 0],
        np.concatenate([np.ones(10), 2.0 * np.ones(8)]),
    )

    # Selected levels are the 0.1 (index 0), 0.5 (index 2), and 0.9 (index 4) columns
    assert quantiles.shape == (2, 8, 3)
    np.testing.assert_array_equal(quantiles[:, :, 0], np.zeros((2, 8)))
    np.testing.assert_array_equal(quantiles[:, :, 1], 2.0 * np.ones((2, 8)))
    np.testing.assert_array_equal(quantiles[:, :, 2], 4.0 * np.ones((2, 8)))
    np.testing.assert_array_equal(mean, 2.0 * np.ones((2, 8)))


def test_timestral_predict_without_covariates() -> None:
    """Test_timestral_predict_without_covariates verifies covariate-free forecasts skip the known rows."""

    class NullForecaster:
        """NullForecaster records the known_covariates argument and returns a minimal Forecast."""

        def __init__(self):
            self.captured_known_covariates: np.ndarray | None = None

        def forecast(self, context, prediction_length, known_covariates=None):
            self.captured_known_covariates = known_covariates
            quantiles = torch.zeros((1, 1, 4, 3), dtype=torch.float32)
            return Forecast(quantiles=quantiles, quantile_levels=(0.1, 0.5, 0.9))

    forecaster = NullForecaster()
    backend = TimestralBackend(model_name="synthetic-test", pipeline=cast(TimestralForecaster, forecaster))

    quantiles, mean = backend.predict(
        target=np.zeros((1, 12), dtype=np.float32),
        past_covariates={},
        future_covariates={},
        prediction_length=4,
    )

    assert forecaster.captured_known_covariates is None
    assert quantiles.shape == (1, 4, 3)
    assert mean.shape == (1, 4)
