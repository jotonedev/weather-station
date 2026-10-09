package readingdata

import (
	"context"
	"time"

	"weather-station/database/repository/timestorage"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Controller holds reading business logic and reads and writes raw weather data.
type Controller struct {
	log  *zap.Logger
	repo *timestorage.Repository
}

type ControllerParams struct {
	fx.In

	Logger *zap.Logger
	Repo   *timestorage.Repository
}

func New(params ControllerParams) (*Controller, error) {
	return &Controller{
		log:  params.Logger,
		repo: params.Repo,
	}, nil
}

// StoreReadings bulk-inserts raw weather readings and returns the number stored.
func (c *Controller) StoreReadings(ctx context.Context, readings []timestorage.Reading) (int64, error) {
	return c.repo.StoreReadings(ctx, readings)
}

// GetReadings returns raw readings for a node within an inclusive time range.
func (c *Controller) GetReadings(ctx context.Context, nodeID string, start, end time.Time) ([]timestorage.Reading, error) {
	return c.repo.GetReadings(ctx, nodeID, start, end)
}

// GetLatestReadings returns the most recent readings for a node.
func (c *Controller) GetLatestReadings(ctx context.Context, nodeID string, limit int) ([]timestorage.Reading, error) {
	return c.repo.GetLatestReadings(ctx, nodeID, limit)
}
