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
		value.Nullable(prediction.RainfallMm), value.Nullable(prediction.RainProbability),
		value.Nullable(prediction.TemperatureCP10), value.Nullable(prediction.TemperatureCP90),
		value.Nullable(prediction.HumidityPctP10), value.Nullable(prediction.HumidityPctP90),
		value.Nullable(prediction.RainfallMmP90),
		value.Nullable(prediction.PressureHpa), value.Nullable(prediction.PressureHpaP10), value.Nullable(prediction.PressureHpaP90),
	}
}

func extractDailyValues(prediction DailyPrediction) []any {
	return []any{
		prediction.NodeID, prediction.Ts, prediction.HorizonDays,
		value.Nullable(prediction.MinTemperatureC), value.Nullable(prediction.MaxTemperatureC),
		value.Nullable(prediction.MinHumidityPct), value.Nullable(prediction.MaxHumidityPct),
		value.Nullable(prediction.RainfallMm), value.Nullable(prediction.RainProbability),
		value.Nullable(prediction.MinTemperatureCP10), value.Nullable(prediction.MaxTemperatureCP90),
		value.Nullable(prediction.RainfallMmP90),
		value.Nullable(prediction.PressureHpa), value.Nullable(prediction.PressureHpaP10), value.Nullable(prediction.PressureHpaP90),
	}
}

// scanHourlyPrediction reads an hourly_predictions row into an HourlyPrediction, using a nil field for a NULL column.
func scanHourlyPrediction(row rowScanner) (HourlyPrediction, error) {
	var prediction HourlyPrediction
	err := row.Scan(
		&prediction.NodeID, &prediction.Ts, &prediction.HorizonHours,
		&prediction.TemperatureC, &prediction.HumidityPct,
		&prediction.RainfallMm, &prediction.RainProbability,
		&prediction.TemperatureCP10, &prediction.TemperatureCP90,
		&prediction.HumidityPctP10, &prediction.HumidityPctP90,
		&prediction.RainfallMmP90,
		&prediction.PressureHpa, &prediction.PressureHpaP10, &prediction.PressureHpaP90,
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
		&prediction.RainfallMm, &prediction.RainProbability,
		&prediction.MinTemperatureCP10, &prediction.MaxTemperatureCP90,
		&prediction.RainfallMmP90,
		&prediction.PressureHpa, &prediction.PressureHpaP10, &prediction.PressureHpaP90,
	)
	return prediction, err
}
