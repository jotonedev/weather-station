package prediction

import "weather-station/common/value"

// predictionSource adapts a []Prediction into a pgx.CopyFromSource.
type predictionSource struct {
	predictions []Prediction
	idx         int
}

func (s *predictionSource) Next() bool {
	if s.idx >= len(s.predictions) {
		return false
	}
	s.idx++
	return true
}

func (s *predictionSource) Values() ([]any, error) {
	prediction := s.predictions[s.idx-1]
	return []any{
		prediction.NodeID, prediction.Ts, prediction.HorizonHours,
		value.Nullable(prediction.RainfallMm), value.Nullable(prediction.TemperatureC), value.Nullable(prediction.HumidityPct), value.Nullable(prediction.WindDirectionDeg),
		value.Nullable(prediction.RainfallConfidence), value.Nullable(prediction.TemperatureConfidence), value.Nullable(prediction.HumidityConfidence), value.Nullable(prediction.WindDirectionConfidence),
	}, nil
}

func (s *predictionSource) Err() error { return nil }
