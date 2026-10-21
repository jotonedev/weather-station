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
		value.Nullable(prediction.RainfallMm), value.Nullable(prediction.TemperatureC),
		value.Nullable(prediction.MinTemperatureC), value.Nullable(prediction.MaxTemperatureC),
		value.Nullable(prediction.HumidityPct), value.Nullable(prediction.MinHumidityPct), value.Nullable(prediction.MaxHumidityPct),
		value.Nullable(prediction.RainfallConfidence),
	}
}

// scanPrediction reads a predictions row into a Prediction, using a nil field for a NULL column.
func scanPrediction(row rowScanner) (Prediction, error) {
	var prediction Prediction
	err := row.Scan(
		&prediction.NodeID, &prediction.Ts, &prediction.HorizonHours,
		&prediction.RainfallMm, &prediction.TemperatureC,
		&prediction.MinTemperatureC, &prediction.MaxTemperatureC,
		&prediction.HumidityPct, &prediction.MinHumidityPct, &prediction.MaxHumidityPct,
		&prediction.RainfallConfidence,
	)
	return prediction, err
}
