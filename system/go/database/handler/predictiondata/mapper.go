package predictiondata

import (
	"weather-station/common/pb"
	"weather-station/database/repository/prediction"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

// pbPredictionToModel converts a proto Prediction into the repository model.
func pbPredictionToModel(p *protodatabasev1.Prediction) prediction.Prediction {
	return prediction.Prediction{
		NodeID:             p.NodeId,
		Ts:                 pb.FromTime(p.IssueTime),
		HorizonHours:       p.HorizonHours,
		RainfallMm:         p.RainfallMm,
		TemperatureC:       p.TemperatureC,
		MinTemperatureC:    p.MinTemperatureC,
		MaxTemperatureC:    p.MaxTemperatureC,
		HumidityPct:        p.HumidityPct,
		MinHumidityPct:     p.MinHumidityPct,
		MaxHumidityPct:     p.MaxHumidityPct,
		RainfallConfidence: p.RainfallConfidence,
	}
}

// modelPredictionToPB converts a repository Prediction model into a proto Prediction.
func modelPredictionToPB(p prediction.Prediction) *protodatabasev1.Prediction {
	return &protodatabasev1.Prediction{
		NodeId:             p.NodeID,
		IssueTime:          pb.Time(p.Ts),
		HorizonHours:       p.HorizonHours,
		RainfallMm:         p.RainfallMm,
		TemperatureC:       p.TemperatureC,
		MinTemperatureC:    p.MinTemperatureC,
		MaxTemperatureC:    p.MaxTemperatureC,
		HumidityPct:        p.HumidityPct,
		MinHumidityPct:     p.MinHumidityPct,
		MaxHumidityPct:     p.MaxHumidityPct,
		RainfallConfidence: p.RainfallConfidence,
	}
}
