package prediction

import (
	"weather-station/common/value"
)

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
