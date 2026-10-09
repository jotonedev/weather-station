package timestorage

import (
	"context"

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

// StoreReadings bulk-inserts readings into the weather table and returns the
// number of rows written.
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
