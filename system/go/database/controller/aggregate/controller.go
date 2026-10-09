package aggregate

import (
	"context"
	"time"

	"weather-station/database/repository/aggregate"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Controller holds aggregate query business logic over raw weather data.
type Controller struct {
	log  *zap.Logger
	repo *aggregate.Repository
}

type ControllerParams struct {
	fx.In

	Logger *zap.Logger
	Repo   *aggregate.Repository
}

func New(params ControllerParams) (*Controller, error) {
	return &Controller{
		log:  params.Logger,
		repo: params.Repo,
	}, nil
}

// GetClimateAggregates returns climate aggregates per time bucket for a node.
func (c *Controller) GetClimateAggregates(ctx context.Context, nodeID string, start, end time.Time, bucketSeconds uint32) ([]aggregate.ClimateAggregate, error) {
	return c.repo.GetClimateAggregates(ctx, nodeID, start, end, bucketSeconds)
}

// GetAirQualityAggregates returns air quality aggregates per time bucket for a node.
func (c *Controller) GetAirQualityAggregates(ctx context.Context, nodeID string, start, end time.Time, bucketSeconds uint32) ([]aggregate.AirQualityAggregate, error) {
	return c.repo.GetAirQualityAggregates(ctx, nodeID, start, end, bucketSeconds)
}
