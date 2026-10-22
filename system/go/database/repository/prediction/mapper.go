package prediction

import (
	"weather-station/common/value"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func extractHourlyValues(prediction HourlyPrediction) []any {
	return []any{
		prediction.NodeID, prediction.Ts, prediction.HorizonHours,
		value.Nullable(prediction.TemperatureC), value.Nullable(prediction.HumidityPct),
		value.Nullable(prediction.RainfallMm), value.Nullable(prediction.RainfallConfidence),
	}
}

func extractDailyValues(prediction DailyPrediction) []any {
	return []any{
		prediction.NodeID, prediction.Ts, prediction.HorizonDays,
		value.Nullable(prediction.MinTemperatureC), value.Nullable(prediction.MaxTemperatureC),
		value.Nullable(prediction.MinHumidityPct), value.Nullable(prediction.MaxHumidityPct),
		value.Nullable(prediction.RainfallMm), value.Nullable(prediction.RainfallConfidence),
	}
}

// scanHourlyPrediction reads an hourly_predictions row into an HourlyPrediction, using a nil field for a NULL column.
func scanHourlyPrediction(row rowScanner) (HourlyPrediction, error) {
	var prediction HourlyPrediction
	err := row.Scan(
		&prediction.NodeID, &prediction.Ts, &prediction.HorizonHours,
		&prediction.TemperatureC, &prediction.HumidityPct,
		&prediction.RainfallMm, &prediction.RainfallConfidence,
	)
	return prediction, err
}

// scanDailyPrediction reads a daily_predictions row into a DailyPrediction, using a nil field for a NULL column.
func scanDailyPrediction(row rowScanner) (DailyPrediction, error) {
	var prediction DailyPrediction
	err := row.Scan(
		&prediction.NodeID, &prediction.Ts, &prediction.HorizonDays,
		&prediction.MinTemperatureC, &prediction.MaxTemperatureC,
		&prediction.MinHumidityPct, &prediction.MaxHumidityPct,
		&prediction.RainfallMm, &prediction.RainfallConfidence,
	)
	return prediction, err
}
