package predictiondata

import (
	"weather-station/common/pb"
	"weather-station/database/repository/prediction"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

// pbPredictionToModel converts a proto Prediction into the repository model.
func pbPredictionToModel(p *protodatabasev1.Prediction) prediction.Prediction {
	return prediction.Prediction{
		NodeID:                  p.NodeId,
		Ts:                      pb.FromTime(p.IssueTime),
		HorizonHours:            p.HorizonHours,
		RainfallMm:              p.RainfallMm,
		TemperatureC:            p.TemperatureC,
		HumidityPct:             p.HumidityPct,
		WindDirectionDeg:        p.WindDirectionDeg,
		RainfallConfidence:      p.RainfallConfidence,
		TemperatureConfidence:   p.TemperatureConfidence,
		HumidityConfidence:      p.HumidityConfidence,
		WindDirectionConfidence: p.WindDirectionConfidence,
	}
}

// modelPredictionToPB converts a repository Prediction model into a proto Prediction.
func modelPredictionToPB(p prediction.Prediction) *protodatabasev1.Prediction {
	return &protodatabasev1.Prediction{
		NodeId:                  p.NodeID,
		IssueTime:               pb.Time(p.Ts),
		HorizonHours:            p.HorizonHours,
		RainfallMm:              p.RainfallMm,
		TemperatureC:            p.TemperatureC,
		HumidityPct:             p.HumidityPct,
		WindDirectionDeg:        p.WindDirectionDeg,
		RainfallConfidence:      p.RainfallConfidence,
		TemperatureConfidence:   p.TemperatureConfidence,
		HumidityConfidence:      p.HumidityConfidence,
		WindDirectionConfidence: p.WindDirectionConfidence,
	}
}
