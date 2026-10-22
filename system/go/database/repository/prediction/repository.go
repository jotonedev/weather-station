package prediction

import (
	"context"

	duckdb "weather-station/database/client"

	"go.uber.org/zap"
)

// Repository persists forecast predictions to the embedded DuckDB hourly and daily prediction tables.
type Repository struct {
	client *duckdb.Client
	log    *zap.Logger
}

func New(client *duckdb.Client, log *zap.Logger) *Repository {
	return &Repository{client: client, log: log}
}

const insertHourlyPredictionSQL = `INSERT INTO hourly_predictions (
	node_id, ts, horizon_hours,
	temperature_c, humidity_pct,
	rainfall_mm, rainfall_confidence
) VALUES (?, ?, ?, ?, ?, ?, ?);`

const insertDailyPredictionSQL = `INSERT INTO daily_predictions (
	node_id, ts, horizon_days,
	min_temperature_c, max_temperature_c,
	min_humidity_pct, max_humidity_pct,
	rainfall_mm, rainfall_confidence
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`

const deleteHourlyPredictionsSQL = `DELETE FROM hourly_predictions WHERE node_id = ? AND ts = ?;`

const deleteDailyPredictionsSQL = `DELETE FROM daily_predictions WHERE node_id = ? AND ts = ?;`

// StoreHourlyPredictions bulk-inserts hourly predictions and returns the number of rows written.
func (r *Repository) StoreHourlyPredictions(ctx context.Context, predictions []HourlyPrediction) (int64, error) {
	if len(predictions) == 0 {
		return 0, nil
	}

	tx, err := r.client.DB().BeginTx(ctx, nil)
	if err != nil {
		r.log.Error("failed to start transaction for hourly predictions", zap.Error(err))
		return 0, err
	}
	defer tx.Rollback()

	deleteStmt, err := tx.PrepareContext(ctx, deleteHourlyPredictionsSQL)
	if err != nil {
		r.log.Error("failed to prepare delete statement for hourly predictions", zap.Error(err))
		return 0, err
	}
	defer deleteStmt.Close()

	distinctKeys := make(map[string]struct{})
	for _, pred := range predictions {
		key := pred.NodeID + pred.Ts.UTC().String()
		if _, seen := distinctKeys[key]; !seen {
			distinctKeys[key] = struct{}{}
			if _, err := deleteStmt.ExecContext(ctx, pred.NodeID, pred.Ts); err != nil {
				r.log.Error("failed to delete prior hourly predictions", zap.String("node_id", pred.NodeID), zap.Error(err))
				return 0, err
			}
		}
	}

	stmt, err := tx.PrepareContext(ctx, insertHourlyPredictionSQL)
	if err != nil {
		r.log.Error("failed to prepare insert statement for hourly predictions", zap.Error(err))
		return 0, err
	}
	defer stmt.Close()

	var stored int64
	for _, pred := range predictions {
		vals := extractHourlyValues(pred)
		if _, err := stmt.ExecContext(ctx, vals...); err != nil {
			r.log.Error("failed to exec insert for hourly prediction", zap.String("node_id", pred.NodeID), zap.Error(err))
			return 0, err
		}
		stored++
	}

	if err := tx.Commit(); err != nil {
		r.log.Error("failed to commit transaction for hourly predictions", zap.Error(err))
		return 0, err
	}

	r.log.Debug("stored hourly predictions", zap.Int64("stored", stored))
	return stored, nil
}

// StoreDailyPredictions bulk-inserts daily predictions and returns the number of rows written.
func (r *Repository) StoreDailyPredictions(ctx context.Context, predictions []DailyPrediction) (int64, error) {
	if len(predictions) == 0 {
		return 0, nil
	}

	tx, err := r.client.DB().BeginTx(ctx, nil)
	if err != nil {
		r.log.Error("failed to start transaction for daily predictions", zap.Error(err))
		return 0, err
	}
	defer tx.Rollback()

	deleteStmt, err := tx.PrepareContext(ctx, deleteDailyPredictionsSQL)
	if err != nil {
		r.log.Error("failed to prepare delete statement for daily predictions", zap.Error(err))
		return 0, err
	}
	defer deleteStmt.Close()

	distinctKeys := make(map[string]struct{})
	for _, pred := range predictions {
		key := pred.NodeID + pred.Ts.UTC().String()
		if _, seen := distinctKeys[key]; !seen {
			distinctKeys[key] = struct{}{}
			if _, err := deleteStmt.ExecContext(ctx, pred.NodeID, pred.Ts); err != nil {
				r.log.Error("failed to delete prior daily predictions", zap.String("node_id", pred.NodeID), zap.Error(err))
				return 0, err
			}
		}
	}

	stmt, err := tx.PrepareContext(ctx, insertDailyPredictionSQL)
	if err != nil {
		r.log.Error("failed to prepare insert statement for daily predictions", zap.Error(err))
		return 0, err
	}
	defer stmt.Close()

	var stored int64
	for _, pred := range predictions {
		vals := extractDailyValues(pred)
		if _, err := stmt.ExecContext(ctx, vals...); err != nil {
			r.log.Error("failed to exec insert for daily prediction", zap.String("node_id", pred.NodeID), zap.Error(err))
			return 0, err
		}
		stored++
	}

	if err := tx.Commit(); err != nil {
		r.log.Error("failed to commit transaction for daily predictions", zap.Error(err))
		return 0, err
	}

	r.log.Debug("stored daily predictions", zap.Int64("stored", stored))
	return stored, nil
}

const hourlySelectColumns = `node_id, ts, horizon_hours,
	temperature_c, humidity_pct,
	rainfall_mm, rainfall_confidence`

const dailySelectColumns = `node_id, ts, horizon_days,
	min_temperature_c, max_temperature_c,
	min_humidity_pct, max_humidity_pct,
	rainfall_mm, rainfall_confidence`

// getLatestHourlyPredictions returns the latest hourly prediction batch for a node, ordered by horizon.
func (r *Repository) getLatestHourlyPredictions(ctx context.Context, nodeID string) ([]HourlyPrediction, error) {
	query := `SELECT ` + hourlySelectColumns + `
		FROM hourly_predictions
		WHERE node_id = ?
		  AND ts = (SELECT MAX(ts) FROM hourly_predictions WHERE node_id = ?)
		QUALIFY row_number() OVER (PARTITION BY horizon_hours ORDER BY ts DESC) = 1
		ORDER BY horizon_hours ASC`

	rows, err := r.client.DB().QueryContext(ctx, query, nodeID, nodeID)
	if err != nil {
		r.log.Error("failed to query hourly forecast", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var predictions []HourlyPrediction
	for rows.Next() {
		prediction, err := scanHourlyPrediction(rows)
		if err != nil {
			r.log.Error("failed to scan hourly prediction", zap.String("node_id", nodeID), zap.Error(err))
			return nil, err
		}
		predictions = append(predictions, prediction)
	}
	if err := rows.Err(); err != nil {
		r.log.Error("error reading hourly prediction rows", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	return predictions, nil
}

// getLatestDailyPredictions returns the latest daily prediction batch for a node, ordered by horizon.
func (r *Repository) getLatestDailyPredictions(ctx context.Context, nodeID string) ([]DailyPrediction, error) {
	query := `SELECT ` + dailySelectColumns + `
		FROM daily_predictions
		WHERE node_id = ?
		  AND ts = (SELECT MAX(ts) FROM daily_predictions WHERE node_id = ?)
		QUALIFY row_number() OVER (PARTITION BY horizon_days ORDER BY ts DESC) = 1
		ORDER BY horizon_days ASC`

	rows, err := r.client.DB().QueryContext(ctx, query, nodeID, nodeID)
	if err != nil {
		r.log.Error("failed to query daily forecast", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var predictions []DailyPrediction
	for rows.Next() {
		prediction, err := scanDailyPrediction(rows)
		if err != nil {
			r.log.Error("failed to scan daily prediction", zap.String("node_id", nodeID), zap.Error(err))
			return nil, err
		}
		predictions = append(predictions, prediction)
	}
	if err := rows.Err(); err != nil {
		r.log.Error("error reading daily prediction rows", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	return predictions, nil
}

// GetForecast returns the latest hourly and daily forecast predictions for a node.
func (r *Repository) GetForecast(ctx context.Context, nodeID string) (Forecast, error) {
	forecast := Forecast{}

	hourly, err := r.getLatestHourlyPredictions(ctx, nodeID)
	if err != nil {
		return Forecast{}, err
	}
	forecast.Hourly = hourly

	daily, err := r.getLatestDailyPredictions(ctx, nodeID)
	if err != nil {
		return Forecast{}, err
	}
	forecast.Daily = daily

	return forecast, nil
}
