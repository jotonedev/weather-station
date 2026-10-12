package prediction

import (
	"context"

	"weather-station/database/client"

	"go.uber.org/zap"
)

// Repository persists forecast predictions to the embedded DuckDB predictions table.
type Repository struct {
	client *duckdb.Client
	log    *zap.Logger
}

func New(client *duckdb.Client, log *zap.Logger) *Repository {
	return &Repository{client: client, log: log}
}

const insertPredictionSQL = `INSERT INTO predictions (
	node_id, ts, horizon_hours,
	rainfall_mm, temperature_c, humidity_pct, wind_direction_deg,
	rainfall_confidence, temperature_confidence, humidity_confidence, wind_direction_confidence
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

// StorePredictions bulk-inserts predictions and returns the number of rows written.
func (r *Repository) StorePredictions(ctx context.Context, predictions []Prediction) (int64, error) {
	if len(predictions) == 0 {
		return 0, nil
	}

	tx, err := r.client.DB().BeginTx(ctx, nil)
	if err != nil {
		r.log.Error("failed to start transaction for predictions", zap.Error(err))
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, insertPredictionSQL)
	if err != nil {
		r.log.Error("failed to prepare insert statement for predictions", zap.Error(err))
		return 0, err
	}
	defer stmt.Close()

	var stored int64
	for _, pred := range predictions {
		vals := extractValues(pred)
		if _, err := stmt.ExecContext(ctx, vals...); err != nil {
			r.log.Error("failed to exec insert for prediction", zap.String("node_id", pred.NodeID), zap.Error(err))
			return 0, err
		}
		stored++
	}

	if err := tx.Commit(); err != nil {
		r.log.Error("failed to commit transaction for predictions", zap.Error(err))
		return 0, err
	}

	r.log.Debug("stored predictions", zap.Int64("stored", stored))
	return stored, nil
}

// predictionSelectColumns lists the SELECT columns.
var predictionSelectColumns = `node_id, ts, horizon_hours,
	rainfall_mm, temperature_c, humidity_pct, wind_direction_deg,
	rainfall_confidence, temperature_confidence, humidity_confidence, wind_direction_confidence`

// GetForecast returns the forecast predictions for the most recent issue time of a node, by horizon.
func (r *Repository) GetForecast(ctx context.Context, nodeID string) ([]Prediction, error) {
	query := `SELECT ` + predictionSelectColumns + `
		FROM predictions
		WHERE node_id = ? AND ts = (SELECT MAX(ts) FROM predictions WHERE node_id = ?)
		ORDER BY horizon_hours ASC`

	rows, err := r.client.DB().QueryContext(ctx, query, nodeID, nodeID)
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
