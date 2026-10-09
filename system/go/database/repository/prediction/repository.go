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
