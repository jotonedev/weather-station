package prediction

import (
	"weather-station/common/value"

	"github.com/jackc/pgx/v5"
)

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

// scanPrediction reads a predictions row into a Prediction, using a nil field for a NULL column.
func scanPrediction(row pgx.Row) (Prediction, error) {
	var prediction Prediction
	err := row.Scan(
		&prediction.NodeID, &prediction.Ts, &prediction.HorizonHours,
		&prediction.RainfallMm, &prediction.TemperatureC, &prediction.HumidityPct, &prediction.WindDirectionDeg,
		&prediction.RainfallConfidence, &prediction.TemperatureConfidence, &prediction.HumidityConfidence, &prediction.WindDirectionConfidence,
	)
	return prediction, err
}
