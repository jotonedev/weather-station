package timestorage

import (
	commonquestdb "weather-station/common/client/questdb"

	"go.uber.org/zap"
)

type Repository struct {
	db  *commonquestdb.Client
	log *zap.Logger
}

func New(db *commonquestdb.Client, log *zap.Logger) *Repository {
	return &Repository{db: db, log: log}
}
