import numpy as np
from google.protobuf.timestamp_pb2 import Timestamp

from inference.formatter import (
    format_daily_predictions,
    format_hourly_predictions,
)


def test_format_hourly_predictions() -> None:
    """Test_format_hourly_predictions verifies horizon steps, p10/p90 bands, pressure, and rain probability."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    # 4 variates: temp, hum, rain, pressure; 24 horizon steps; 3 quantiles (10%, 50%, 90%)
    quantiles = np.zeros((4, 24, 3), dtype=np.float32)

    quantiles[0, :, 0] = 20.0
    quantiles[0, :, 1] = 22.0
    quantiles[0, :, 2] = 24.0

    quantiles[1, :, 0] = 55.0
    quantiles[1, :, 1] = 60.0
    quantiles[1, :, 2] = 65.0

    quantiles[2, :, 0] = 0.5
    quantiles[2, :, 1] = 1.5
    quantiles[2, :, 2] = 2.5

    quantiles[3, :, 0] = 1010.0
    quantiles[3, :, 1] = 1013.0
    quantiles[3, :, 2] = 1016.0

    target_columns = ["temperature_c", "humidity_pct", "rainfall_mm", "pressure_hpa"]

    predictions = format_hourly_predictions(
        node_id="station-one",
        issue_time=issue_time,
        horizon_hours=24,
        quantiles=quantiles,
        target_columns=target_columns,
        quantile_levels=(0.1, 0.5, 0.9),
    )

    assert len(predictions) == 24
    for step_index, pred in enumerate(predictions):
        assert pred.node_id == "station-one"
        assert pred.horizon_hours == step_index + 1
        assert np.isclose(pred.temperature_c, 22.0)
        assert np.isclose(pred.temperature_c_p10, 20.0)
        assert np.isclose(pred.temperature_c_p90, 24.0)
        assert np.isclose(pred.humidity_pct, 60.0)
        assert np.isclose(pred.humidity_pct_p10, 55.0)
        assert np.isclose(pred.humidity_pct_p90, 65.0)
        assert np.isclose(pred.rainfall_mm, 1.5)
        assert np.isclose(pred.rainfall_mm_p90, 2.5)
        assert np.isclose(pred.rain_probability, 1.0)
        assert np.isclose(pred.pressure_hpa, 1013.0)
        assert np.isclose(pred.pressure_hpa_p10, 1010.0)
        assert np.isclose(pred.pressure_hpa_p90, 1016.0)


def test_format_daily_predictions() -> None:
    """Test_format_daily_predictions verifies daily predictions start at horizon_days=0 with bounds and pressure."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    # 6 variates: min_temp, max_temp, min_hum, max_hum, rain, pressure
    quantiles = np.zeros((6, 10, 3), dtype=np.float32)
    quantiles[0, :, :] = [10.0, 12.0, 14.0]
    quantiles[1, :, :] = [22.0, 24.0, 27.0]
    quantiles[2, :, :] = [35.0, 40.0, 45.0]
    quantiles[3, :, :] = [75.0, 80.0, 85.0]
    quantiles[4, :, :] = [1.0, 2.0, 3.0]
    quantiles[5, :, :] = [1008.0, 1012.0, 1015.0]

    target_columns = [
        "min_temperature_c",
        "max_temperature_c",
        "min_humidity_pct",
        "max_humidity_pct",
        "rainfall_mm",
        "pressure_hpa",
    ]

    predictions = format_daily_predictions(
        node_id="station-two",
        issue_time=issue_time,
        horizon_days=10,
        quantiles=quantiles,
        target_columns=target_columns,
        quantile_levels=(0.1, 0.5, 0.9),
    )

    assert len(predictions) == 10
    actual_horizons = [pred.horizon_days for pred in predictions]
    assert actual_horizons == list(range(10))
    assert np.isclose(predictions[0].min_temperature_c, 12.0)
    assert np.isclose(predictions[0].min_temperature_c_p10, 10.0)
    assert np.isclose(predictions[0].max_temperature_c, 24.0)
    assert np.isclose(predictions[0].max_temperature_c_p90, 27.0)
    assert np.isclose(predictions[0].min_humidity_pct, 40.0)
    assert np.isclose(predictions[0].max_humidity_pct, 80.0)
    assert np.isclose(predictions[0].rainfall_mm, 2.0)
    assert np.isclose(predictions[0].rainfall_mm_p90, 3.0)
    assert np.isclose(predictions[0].rain_probability, 1.0)
    assert np.isclose(predictions[0].pressure_hpa, 1012.0)
    assert np.isclose(predictions[0].pressure_hpa_p10, 1008.0)
    assert np.isclose(predictions[0].pressure_hpa_p90, 1015.0)


def test_format_predictions_clamping() -> None:
    """Test_format_predictions_clamping ensures physical constraints on output values."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    quantiles = np.zeros((3, 1, 3), dtype=np.float32)
    quantiles[0, 0, :] = -5.0
    quantiles[1, 0, :] = -120.0
    quantiles[2, 0, :] = 150.0

    target_columns = ["rainfall_mm", "temperature_c", "humidity_pct"]

    predictions = format_hourly_predictions(
        node_id="station-three",
        issue_time=issue_time,
        horizon_hours=1,
        quantiles=quantiles,
        target_columns=target_columns,
        quantile_levels=(0.1, 0.5, 0.9),
    )

    assert predictions[0].rainfall_mm == 0.0
    assert predictions[0].rain_probability == 0.0
    assert predictions[0].temperature_c == -90.0
    assert predictions[0].humidity_pct == 100.0


def test_format_predictions_retains_sub_threshold_drizzle() -> None:
    """Test_format_predictions_retains_sub_threshold_drizzle keeps positive median rainfall below 0.2 mm."""
    issue_time = Timestamp()
    issue_time.FromSeconds(1700000000)

    quantiles = np.zeros((1, 1, 3), dtype=np.float32)
    quantiles[0, 0, :] = [0.0, 0.1, 0.15]

    predictions = format_hourly_predictions(
        node_id="station-drizzle",
        issue_time=issue_time,
        horizon_hours=1,
        quantiles=quantiles,
        target_columns=["rainfall_mm"],
        quantile_levels=(0.1, 0.5, 0.9),
    )

    assert np.isclose(predictions[0].rainfall_mm, 0.1)
    assert np.isclose(predictions[0].rainfall_mm_p90, 0.15)
    assert predictions[0].rain_probability == 0.0
