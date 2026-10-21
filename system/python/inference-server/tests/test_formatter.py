import numpy as np
from google.protobuf.timestamp_pb2 import Timestamp

from inference.formatter import (
    compute_confidence,
    format_daily_predictions,
    format_hourly_predictions,
)


def test_compute_confidence() -> None:
    """Test_compute_confidence validates range clamping and scaling."""
    assert compute_confidence(0.0, 10.0) == 1.0
    assert compute_confidence(5.0, 10.0) == 0.5
    assert compute_confidence(15.0, 10.0) == 0.0
    assert compute_confidence(-2.0, 10.0) == 1.0


def test_format_hourly_predictions() -> None:
    """Test_format_hourly_predictions verifies horizon steps and field population for hourly forecast."""
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

    predictions = format_hourly_predictions(
        node_id="station-one",
        issue_time=issue_time,
        horizon_hours=24,
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
        assert 0.0 <= pred.rainfall_confidence <= 1.0


def test_format_daily_predictions() -> None:
    """Test_format_daily_predictions verifies daily predictions with min/max temp and humidity."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    # 5 variates: min_temp, max_temp, min_hum, max_hum, rain
    quantiles = np.zeros((5, 10, 3), dtype=np.float32)
    # min_temperature_c
    quantiles[0, :, 1] = 12.0
    # max_temperature_c
    quantiles[1, :, 1] = 24.0
    # min_humidity_pct
    quantiles[2, :, 1] = 40.0
    # max_humidity_pct
    quantiles[3, :, 1] = 80.0
    # rainfall_mm
    quantiles[4, :, 1] = 2.0

    target_columns = [
        "min_temperature_c",
        "max_temperature_c",
        "min_humidity_pct",
        "max_humidity_pct",
        "rainfall_mm",
    ]

    predictions = format_daily_predictions(
        node_id="station-two",
        issue_time=issue_time,
        horizon_days=10,
        quantiles=quantiles,
        target_columns=target_columns,
    )

    assert len(predictions) == 10
    actual_horizons = [pred.horizon_days for pred in predictions]
    assert actual_horizons == list(range(1, 11))
    assert np.isclose(predictions[0].min_temperature_c, 12.0)
    assert np.isclose(predictions[0].max_temperature_c, 24.0)
    assert np.isclose(predictions[0].min_humidity_pct, 40.0)
    assert np.isclose(predictions[0].max_humidity_pct, 80.0)
    assert np.isclose(predictions[0].rainfall_mm, 2.0)


def test_format_predictions_clamping() -> None:
    """Test_format_predictions_clamping ensures physical constraints on output values."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    quantiles = np.zeros((3, 1, 3), dtype=np.float32)
    # Negative rainfall -5mm, temp -120C (clamped to -90), hum 150% (clamped to 100)
    quantiles[0, 0, 1] = -5.0
    quantiles[1, 0, 1] = -120.0
    quantiles[2, 0, 1] = 150.0

    target_columns = ["rainfall_mm", "temperature_c", "humidity_pct"]

    predictions = format_hourly_predictions(
        node_id="station-three",
        issue_time=issue_time,
        horizon_hours=1,
        quantiles=quantiles,
        target_columns=target_columns,
    )

    assert predictions[0].rainfall_mm == 0.0
    assert predictions[0].temperature_c == -90.0
    assert predictions[0].humidity_pct == 100.0
