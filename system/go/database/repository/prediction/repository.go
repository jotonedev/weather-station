package prediction

import (
	"context"

	commonquestdb "weather-station/common/client/questdb"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// Repository persists forecast predictions to the QuestDB predictions table.
type Repository struct {
	db  *commonquestdb.Client
	log *zap.Logger
}

func New(db *commonquestdb.Client, log *zap.Logger) *Repository {
	return &Repository{db: db, log: log}
}

var predictionTable = pgx.Identifier{"predictions"}

// StorePredictions bulk-inserts predictions and returns the number of rows written.
func (r *Repository) StorePredictions(ctx context.Context, predictions []Prediction) (int64, error) {
	if len(predictions) == 0 {
		return 0, nil
	}
	stored, err := r.db.CopyFrom(ctx, predictionTable, predictionColumns, &predictionSource{predictions: predictions})
	if err != nil {
		r.log.Error("failed to store predictions", zap.Error(err), zap.Int("predictions", len(predictions)))
		return 0, err
	}
	r.log.Debug("stored predictions", zap.Int64("stored", stored))
	return stored, nil
}

// predictionColumns lists the SELECT columns, mirroring predictionColumns order.
var predictionSelectColumns = `node_id, ts, horizon_hours,
	rainfall_mm, temperature_c, humidity_pct, wind_direction_deg,
	rainfall_confidence, temperature_confidence, humidity_confidence, wind_direction_confidence`

// GetForecast returns the forecast predictions for the most recent issue time of a node, by horizon.
func (r *Repository) GetForecast(ctx context.Context, nodeID string) ([]Prediction, error) {
	query := `SELECT ` + predictionSelectColumns + `
		FROM predictions
		WHERE node_id = $1 AND ts = (SELECT MAX(ts) FROM predictions WHERE node_id = $1)
		ORDER BY horizon_hours ASC`

	rows, err := r.db.Query(ctx, query, nodeID)
	if err != nil {
		r.log.Error("failed to query forecast", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var predictions []Prediction
	for rows.Next() {
		prediction, err := scanPrediction(rows)
		if err != nil {
			r.log.Error("failed to scan prediction", zap.String("node_id", nodeID), zap.Error(err))
			return nil, err
		}
		predictions = append(predictions, prediction)
	}
	if err := rows.Err(); err != nil {
		r.log.Error("error reading prediction rows", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	return predictions, nil
}
