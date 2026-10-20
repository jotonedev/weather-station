import numpy as np
from google.protobuf.timestamp_pb2 import Timestamp
from weather.inference.v1.inference_pb2 import (
    PREDICTION_MODE_NEXT_10_DAYS,
    PREDICTION_MODE_NEXT_24_HOURS,
)

from inference.formatter import compute_confidence, format_predictions


def test_compute_confidence() -> None:
    """Test_compute_confidence validates range clamping and scaling."""
    assert compute_confidence(0.0, 10.0) == 1.0
    assert compute_confidence(5.0, 10.0) == 0.5
    assert compute_confidence(15.0, 10.0) == 0.0
    assert compute_confidence(-2.0, 10.0) == 1.0


def test_format_predictions_hourly_mode() -> None:
    """Test_format_predictions_hourly_mode verifies horizon steps and field population for 24 hour forecast."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    # 3 variates: temp, hum, rain
    # 24 horizon steps, 3 quantiles (10%, 50%, 90%)
    quantiles = np.zeros((3, 24, 3), dtype=np.float32)

    # Temperature: median 22.0, spread 4.0
    quantiles[0, :, 0] = 20.0
    quantiles[0, :, 1] = 22.0
    quantiles[0, :, 2] = 24.0

    # Humidity: median 60.0, spread 10.0
    quantiles[1, :, 0] = 55.0
    quantiles[1, :, 1] = 60.0
    quantiles[1, :, 2] = 65.0

    # Rainfall: median 1.5, spread 2.0
    quantiles[2, :, 0] = 0.5
    quantiles[2, :, 1] = 1.5
    quantiles[2, :, 2] = 2.5

    target_columns = ["temperature_c", "humidity_pct", "rainfall_mm"]

    predictions = format_predictions(
        node_id="station-one",
        issue_time=issue_time,
        mode=PREDICTION_MODE_NEXT_24_HOURS,
        quantiles=quantiles,
        target_columns=target_columns,
    )

    assert len(predictions) == 24
    for step_index, pred in enumerate(predictions):
        assert pred.node_id == "station-one"
        assert pred.horizon_hours == step_index + 1
        assert np.isclose(pred.temperature_c, 22.0)
        assert np.isclose(pred.humidity_pct, 60.0)
        assert np.isclose(pred.rainfall_mm, 1.5)
        assert not pred.HasField("wind_direction_deg")
        assert 0.0 <= pred.temperature_confidence <= 1.0
        assert 0.0 <= pred.humidity_confidence <= 1.0


def test_format_predictions_daily_mode() -> None:
    """Test_format_predictions_daily_mode verifies 10 daily predictions spaced by 24 hours."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    quantiles = np.zeros((2, 10, 3), dtype=np.float32)
    quantiles[0, :, 1] = 18.0
    quantiles[1, :, 1] = 45.0

    target_columns = ["temperature_c", "humidity_pct"]

    predictions = format_predictions(
        node_id="station-two",
        issue_time=issue_time,
        mode=PREDICTION_MODE_NEXT_10_DAYS,
        quantiles=quantiles,
        target_columns=target_columns,
    )

    assert len(predictions) == 10
    expected_horizons = [step * 24 for step in range(1, 11)]
    actual_horizons = [pred.horizon_hours for pred in predictions]
    assert actual_horizons == expected_horizons


def test_format_predictions_clamping() -> None:
    """Test_format_predictions_clamping ensures physical constraints on output values."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    quantiles = np.zeros((3, 1, 3), dtype=np.float32)
    # Humidity out of range 120%
    quantiles[1, 0, 1] = 120.0
    # Negative rainfall -5mm
    quantiles[2, 0, 1] = -5.0

    target_columns = ["temperature_c", "humidity_pct", "rainfall_mm"]

    predictions = format_predictions(
        node_id="station-three",
        issue_time=issue_time,
        mode=PREDICTION_MODE_NEXT_24_HOURS,
        quantiles=quantiles,
        target_columns=target_columns,
    )

    assert predictions[0].humidity_pct == 100.0
    assert predictions[0].rainfall_mm == 0.0
