package prediction

import (
	"weather-station/common/value"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func extractValues(prediction Prediction) []any {
	return []any{
		prediction.NodeID, prediction.Ts, prediction.HorizonHours,
		value.Nullable(prediction.RainfallMm), value.Nullable(prediction.TemperatureC), value.Nullable(prediction.HumidityPct), value.Nullable(prediction.WindDirectionDeg),
		value.Nullable(prediction.RainfallConfidence), value.Nullable(prediction.TemperatureConfidence), value.Nullable(prediction.HumidityConfidence), value.Nullable(prediction.WindDirectionConfidence),
	}
}

// scanPrediction reads a predictions row into a Prediction, using a nil field for a NULL column.
func scanPrediction(row rowScanner) (Prediction, error) {
	var prediction Prediction
	err := row.Scan(
		&prediction.NodeID, &prediction.Ts, &prediction.HorizonHours,
		&prediction.RainfallMm, &prediction.TemperatureC, &prediction.HumidityPct, &prediction.WindDirectionDeg,
		&prediction.RainfallConfidence, &prediction.TemperatureConfidence, &prediction.HumidityConfidence, &prediction.WindDirectionConfidence,
	)
	return prediction, err
}
