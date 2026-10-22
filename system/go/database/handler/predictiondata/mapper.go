package predictiondata

import (
	"weather-station/common/pb"
	"weather-station/database/repository/prediction"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

// pbHourlyToModel converts a proto HourlyPrediction into the repository model.
func pbHourlyToModel(p *protodatabasev1.HourlyPrediction) prediction.HourlyPrediction {
	return prediction.HourlyPrediction{
		NodeID:             p.NodeId,
		Ts:                 pb.FromTime(p.IssueTime),
		HorizonHours:       p.HorizonHours,
		TemperatureC:       p.TemperatureC,
		HumidityPct:        p.HumidityPct,
		RainfallMm:         p.RainfallMm,
		RainfallConfidence: p.RainfallConfidence,
	}
}

// pbDailyToModel converts a proto DailyPrediction into the repository model.
func pbDailyToModel(p *protodatabasev1.DailyPrediction) prediction.DailyPrediction {
	return prediction.DailyPrediction{
		NodeID:             p.NodeId,
		Ts:                 pb.FromTime(p.IssueTime),
		HorizonDays:        p.HorizonDays,
		MinTemperatureC:    p.MinTemperatureC,
		MaxTemperatureC:    p.MaxTemperatureC,
		MinHumidityPct:     p.MinHumidityPct,
		MaxHumidityPct:     p.MaxHumidityPct,
		RainfallMm:         p.RainfallMm,
		RainfallConfidence: p.RainfallConfidence,
	}
}

// modelHourlyToPB converts a repository HourlyPrediction model into a proto HourlyPrediction.
func modelHourlyToPB(p prediction.HourlyPrediction) *protodatabasev1.HourlyPrediction {
	return &protodatabasev1.HourlyPrediction{
		NodeId:             p.NodeID,
		IssueTime:          pb.Time(p.Ts),
		HorizonHours:       p.HorizonHours,
		TemperatureC:       p.TemperatureC,
		HumidityPct:        p.HumidityPct,
		RainfallMm:         p.RainfallMm,
		RainfallConfidence: p.RainfallConfidence,
	}
}

// modelDailyToPB converts a repository DailyPrediction model into a proto DailyPrediction.
func modelDailyToPB(p prediction.DailyPrediction) *protodatabasev1.DailyPrediction {
	return &protodatabasev1.DailyPrediction{
		NodeId:             p.NodeID,
		IssueTime:          pb.Time(p.Ts),
		HorizonDays:        p.HorizonDays,
		MinTemperatureC:    p.MinTemperatureC,
		MaxTemperatureC:    p.MaxTemperatureC,
		MinHumidityPct:     p.MinHumidityPct,
		MaxHumidityPct:     p.MaxHumidityPct,
		RainfallMm:         p.RainfallMm,
		RainfallConfidence: p.RainfallConfidence,
	}
}
