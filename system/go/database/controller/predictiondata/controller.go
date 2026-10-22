package predictiondata

import (
	"context"

	"weather-station/database/repository/prediction"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Controller struct {
	log  *zap.Logger
	repo *prediction.Repository
}

type ControllerParams struct {
	fx.In

	Logger *zap.Logger
	Repo   *prediction.Repository
}

func New(params ControllerParams) (*Controller, error) {
	return &Controller{
		log:  params.Logger,
		repo: params.Repo,
	}, nil
}

// StoreHourlyPredictions bulk-inserts hourly forecast predictions and returns the number stored.
func (c *Controller) StoreHourlyPredictions(ctx context.Context, predictions []prediction.HourlyPrediction) (int64, error) {
	return c.repo.StoreHourlyPredictions(ctx, predictions)
}

// StoreDailyPredictions bulk-inserts daily forecast predictions and returns the number stored.
func (c *Controller) StoreDailyPredictions(ctx context.Context, predictions []prediction.DailyPrediction) (int64, error) {
	return c.repo.StoreDailyPredictions(ctx, predictions)
}

// GetForecast returns the latest hourly and daily forecast predictions for a node.
func (c *Controller) GetForecast(ctx context.Context, nodeID string) (prediction.Forecast, error) {
	return c.repo.GetForecast(ctx, nodeID)
}
