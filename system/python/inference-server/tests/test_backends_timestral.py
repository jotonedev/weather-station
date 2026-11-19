from typing import cast

import numpy as np
import pytest

from inference.backends.base import _split_covariates
from tests.helpers import create_synthetic_timestral_forecaster


def test_split_covariates_separates_past_only_and_known() -> None:
    """Test_split_covariates_separates_past_only_and_known verifies past-only and known tensor layouts."""
    past = {
        "dew_point_c": np.array([10.0, 11.0]),
        "solar_hour_sin": np.array([1.0, 2.0]),
        "solar_hour_cos": np.array([3.0, 4.0]),
    }
    future = {
        "solar_hour_sin": np.array([5.0]),
        "solar_hour_cos": np.array([6.0]),
    }

    past_tensor, known_tensor = _split_covariates(past, future)

    assert past_tensor is not None
    assert past_tensor.shape == (1, 1, 2)
    np.testing.assert_array_equal(past_tensor[0, 0], [10.0, 11.0])

    assert known_tensor is not None
    assert known_tensor.shape == (1, 2, 3)
    assert known_tensor.dtype == np.float32
    np.testing.assert_array_equal(known_tensor[0, 0], [1.0, 2.0, 5.0])
    np.testing.assert_array_equal(known_tensor[0, 1], [3.0, 4.0, 6.0])


def test_split_covariates_single_side_only() -> None:
    """Test_split_covariates_single_side_only verifies past-only and known-only channel tensors."""
    past_only_tensor, known_when_past_only = _split_covariates(
        {"dew_point_c": np.array([10.0, 11.0])},
        {},
    )
    assert past_only_tensor is not None
    assert past_only_tensor.shape == (1, 1, 2)
    assert past_only_tensor.dtype == np.float32
    assert known_when_past_only is None

    past_when_known_only, known_only_tensor = _split_covariates(
        {"solar_hour_sin": np.array([1.0, 2.0])},
        {"solar_hour_sin": np.array([3.0])},
    )
    assert past_when_known_only is None
    assert known_only_tensor is not None
    assert known_only_tensor.shape == (1, 1, 3)
    np.testing.assert_array_equal(known_only_tensor[0, 0], [1.0, 2.0, 3.0])


def test_split_covariates_missing_past_keys_raises() -> None:
    """Test_split_covariates_missing_past_keys_raises verifies future keys without past counterparts fail."""
    with pytest.raises(ValueError, match="missing keys"):
        _split_covariates({}, {"solar_hour_sin": np.array([1.0])})


@pytest.fixture
def timestral_package():
    """Timestral_package skips optional Timestral tests when timestral is not installed."""
    return pytest.importorskip("timestral")


def test_timestral_predict_with_synthetic_model(timestral_package: object) -> None:
    """Test_timestral_predict_with_synthetic_model verifies a full CPU forecast and unbounded horizon."""
    from inference.backends.timestral import TimestralBackend

    forecaster = create_synthetic_timestral_forecaster()
    backend = TimestralBackend(model_name="synthetic-test", pipeline=forecaster)
    assert backend.max_recommended_horizon() is None

    history_length = 40
    prediction_length = 16
    rng = np.random.default_rng(0)
    target = rng.standard_normal((3, history_length)).astype(np.float32)
    known_names = ("solar_hour_sin", "solar_hour_cos", "doy_sin", "doy_cos")
    past_covariates = {name: rng.standard_normal(history_length).astype(np.float32) for name in known_names}
    past_covariates["dew_point_c"] = rng.standard_normal(history_length).astype(np.float32)
    future_covariates = {name: rng.standard_normal(prediction_length).astype(np.float32) for name in known_names}

    quantiles, levels = backend.predict(
        target=target,
        past_covariates=past_covariates,
        future_covariates=future_covariates,
        prediction_length=prediction_length,
    )

    assert quantiles.shape == (3, prediction_length, 9)
    assert len(levels) == 9
    assert np.isfinite(quantiles).all()
    assert (np.diff(quantiles, axis=-1) >= -1e-5).all()


@pytest.mark.parametrize(
    ("with_covariates", "symmetric_averaging"),
    [
        (True, False),
        (False, True),
    ],
)
def test_timestral_predict_maps_inputs_and_all_levels(
    timestral_package: object,
    with_covariates: bool,
    symmetric_averaging: bool,
) -> None:
    """Test_timestral_predict_maps_inputs_and_all_levels verifies context, covariates, and knot output."""
    import torch
    from timestral import Forecast, TimestralForecaster

    from inference.backends.timestral import TimestralBackend

    class RecordingForecaster:
        """RecordingForecaster records forecast arguments and returns a fixed Forecast object."""

        def __init__(self):
            self.captured_context = None
            self.captured_prediction_length = None
            self.captured_past_covariates: np.ndarray | None = None
            self.captured_known_covariates: np.ndarray | None = None
            self.captured_clip_nonnegative: bool | None = None
            self.captured_symmetric_averaging: bool | None = None
            self.quantile_levels = (0.1, 0.3, 0.5, 0.7, 0.9)

        def forecast(
            self,
            context,
            prediction_length,
            past_covariates=None,
            known_covariates=None,
            clip_nonnegative=True,
            symmetric_averaging=True,
        ):
            """forecast captures arguments and returns quantiles holding the level index at every position."""
            self.captured_context = context
            self.captured_prediction_length = prediction_length
            self.captured_past_covariates = past_covariates
            self.captured_known_covariates = known_covariates
            self.captured_clip_nonnegative = clip_nonnegative
            self.captured_symmetric_averaging = symmetric_averaging
            quantiles = torch.arange(5, dtype=torch.float32).reshape(1, 1, 1, 5).expand(1, 2, 8, 5).clone()
            return Forecast(quantiles=quantiles, quantile_levels=self.quantile_levels)

    forecaster = RecordingForecaster()
    backend = TimestralBackend(
        model_name="synthetic-test",
        pipeline=cast(TimestralForecaster, forecaster),
        symmetric_averaging=symmetric_averaging,
    )

    target = np.zeros((2, 10), dtype=np.float32)
    if with_covariates:
        past_covariates = {
            "wind_gust_mps": 3.0 * np.ones(10, dtype=np.float32),
            "solar_hour_sin": np.ones(10, dtype=np.float32),
        }
        future_covariates = {"solar_hour_sin": 2.0 * np.ones(8, dtype=np.float32)}
    else:
        past_covariates = {}
        future_covariates = {}

    quantiles, levels = backend.predict(
        target=target,
        past_covariates=past_covariates,
        future_covariates=future_covariates,
        prediction_length=8,
    )

    assert forecaster.captured_prediction_length == 8
    np.testing.assert_array_equal(forecaster.captured_context, target[None, ...])
    assert forecaster.captured_clip_nonnegative is False
    assert forecaster.captured_symmetric_averaging is symmetric_averaging
    assert quantiles.shape == (2, 8, 5)
    assert levels == (0.1, 0.3, 0.5, 0.7, 0.9)

    if with_covariates:
        past_tensor = forecaster.captured_past_covariates
        assert past_tensor is not None
        assert past_tensor.shape == (1, 1, 10)
        np.testing.assert_array_equal(past_tensor[0, 0], 3.0 * np.ones(10))

        known_covariates = forecaster.captured_known_covariates
        assert known_covariates is not None
        assert known_covariates.shape == (1, 1, 18)
        np.testing.assert_array_equal(
            known_covariates[0, 0],
            np.concatenate([np.ones(10), 2.0 * np.ones(8)]),
        )
    else:
        assert forecaster.captured_past_covariates is None
        assert forecaster.captured_known_covariates is None
