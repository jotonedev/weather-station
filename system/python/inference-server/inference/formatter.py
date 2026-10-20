import numpy as np
from google.protobuf.timestamp_pb2 import Timestamp
from weather.database.v1.prediction_pb2 import Prediction
from weather.inference.v1.inference_pb2 import (
    PREDICTION_MODE_NEXT_10_DAYS,
    PREDICTION_MODE_NEXT_24_HOURS,
)


def compute_confidence(spread: float, scale: float) -> float:
    """compute_confidence translates prediction interval spread to a normalized 0 to 1 confidence."""
    confidence = 1.0 - (max(0.0, spread) / scale)
    return float(np.clip(confidence, 0.0, 1.0))


def format_predictions(
    node_id: str,
    issue_time: Timestamp,
    mode: int,
    quantiles: np.ndarray,
    target_columns: list[str],
) -> list[Prediction]:
    """format_predictions converts model quantile forecasts into database prediction records."""
    if mode == PREDICTION_MODE_NEXT_24_HOURS:
        horizon_hours_list = list(range(1, 25))
    elif mode == PREDICTION_MODE_NEXT_10_DAYS:
        horizon_hours_list = [step * 24 for step in range(1, 11)]
    else:
        return []

    var_indices = {col: idx for idx, col in enumerate(target_columns)}
    predictions = []
    total_steps = min(len(horizon_hours_list), quantiles.shape[1])

    for step_idx in range(total_steps):
        horizon_hour = horizon_hours_list[step_idx]
        prediction = Prediction(
            node_id=node_id,
            issue_time=issue_time,
            horizon_hours=horizon_hour,
        )

        # Temperature
        if "temperature_c" in var_indices:
            idx = var_indices["temperature_c"]
            temp_med = float(np.clip(quantiles[idx, step_idx, 1], -90.0, 70.0))
            temp_spread = float(quantiles[idx, step_idx, 2] - quantiles[idx, step_idx, 0])
            prediction.temperature_c = temp_med
            prediction.temperature_confidence = compute_confidence(temp_spread, 20.0)

        # Humidity
        if "humidity_pct" in var_indices:
            idx = var_indices["humidity_pct"]
            hum_med = float(np.clip(quantiles[idx, step_idx, 1], 0.0, 100.0))
            hum_spread = float(quantiles[idx, step_idx, 2] - quantiles[idx, step_idx, 0])
            prediction.humidity_pct = hum_med
            prediction.humidity_confidence = compute_confidence(hum_spread, 50.0)

        # Rainfall
        if "rainfall_mm" in var_indices:
            idx = var_indices["rainfall_mm"]
            rain_med = float(max(0.0, quantiles[idx, step_idx, 1]))
            rain_spread = float(quantiles[idx, step_idx, 2] - quantiles[idx, step_idx, 0])
            prediction.rainfall_mm = rain_med
            prediction.rainfall_confidence = compute_confidence(rain_spread, 20.0)

        predictions.append(prediction)

    return predictions
