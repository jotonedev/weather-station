package timestorage

import (
	"context"
	"time"

	commonquestdb "weather-station/common/client/questdb"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// Repository persists raw weather readings to the QuestDB weather table.
type Repository struct {
	db  *commonquestdb.Client
	log *zap.Logger
}

func New(db *commonquestdb.Client, log *zap.Logger) *Repository {
	return &Repository{db: db, log: log}
}

// weatherTable is the target table for CopyFrom bulk ingest.
var weatherTable = pgx.Identifier{"weather"}

// StoreReadings bulk-inserts readings into the weather table and returns the number of rows written.
func (r *Repository) StoreReadings(ctx context.Context, readings []Reading) (int64, error) {
	if len(readings) == 0 {
		return 0, nil
	}
	stored, err := r.db.CopyFrom(ctx, weatherTable, weatherColumns, &readingSource{readings: readings})
	if err != nil {
		r.log.Error("failed to store weather readings", zap.Error(err), zap.Int("readings", len(readings)))
		return 0, err
	}
	r.log.Debug("stored weather readings", zap.Int64("stored", stored))
	return stored, nil
}

// readingColumns lists the SELECT columns, mirroring weatherColumns order.
var readingColumns = `node_id, ts,
	wind_speed, wind_angle, raw_wind_speed, raw_wind_angle,
	temperature, humidity, pressure,
	rain_1h, rain_24h, rain_rate,
	pm1_0, pm2_5, pm10_0, voc,
	anemo_status, anemo_agc, anemo_agc_variance,
	vane_status, vane_agc, vane_agc_variance,
	rain_status, rain_agc, rain_agc_variance`

// GetReadings returns raw readings for a node within an inclusive time range, oldest first.
func (r *Repository) GetReadings(ctx context.Context, nodeID string, start, end time.Time) ([]Reading, error) {
	query := `SELECT ` + readingColumns + `
		FROM weather WHERE node_id = $1 AND ts >= $2 AND ts <= $3
		ORDER BY ts ASC`

	rows, err := r.db.Query(ctx, query, nodeID, start, end)
	if err != nil {
		r.log.Error("failed to query weather readings", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var readings []Reading
	for rows.Next() {
		reading, err := scanReading(rows)
		if err != nil {
			r.log.Error("failed to scan weather reading", zap.String("node_id", nodeID), zap.Error(err))
			return nil, err
		}
		readings = append(readings, reading)
	}
	if err := rows.Err(); err != nil {
		r.log.Error("error reading weather rows", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	return readings, nil
}

// GetLatestReadings returns the most recent raw readings for a node, newest first, up to limit rows.
func (r *Repository) GetLatestReadings(ctx context.Context, nodeID string, limit int) ([]Reading, error) {
	query := `SELECT ` + readingColumns + `
		FROM weather WHERE node_id = $1
		ORDER BY ts DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, query, nodeID, limit)
	if err != nil {
		r.log.Error("failed to query latest weather readings", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var readings []Reading
	for rows.Next() {
		reading, err := scanReading(rows)
		if err != nil {
			r.log.Error("failed to scan latest weather reading", zap.String("node_id", nodeID), zap.Error(err))
			return nil, err
		}
		readings = append(readings, reading)
	}
	if err := rows.Err(); err != nil {
		r.log.Error("error reading latest weather rows", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	return readings, nil
}
