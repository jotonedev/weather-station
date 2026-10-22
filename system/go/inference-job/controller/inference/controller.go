package inference

//go:generate mockgen -source controller.go -destination ../../internal/mocks/gateways.go -package mocks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
	protoinferencev1 "weather-station/gen/go/weather/inference/v1"
	"weather-station/inference-job/gateway/database"
	"weather-station/inference-job/gateway/inferenceserver"
)

var (
	// ErrInvalidHorizon is returned when an invalid prediction horizon is supplied.
	ErrInvalidHorizon = errors.New("invalid prediction horizon")
	// ErrNoGateways is returned when required gateways are missing.
	ErrNoGateways = errors.New("database and inference gateways are required")
)

// DatabaseGateway abstracts database operations needed by the inference controller.
type DatabaseGateway interface {
	ListNodeCoordinates(ctx context.Context) ([]*protodatabasev1.NodeCoordinates, error)
	GetClimateAggregates(ctx context.Context, nodeID string, start, end time.Time, bucketSeconds uint32) ([]*protodatabasev1.ClimateAggregate, error)
	StorePredictions(ctx context.Context, predictions []*protodatabasev1.Prediction) error
}

// InferenceGateway abstracts inference server operations needed by the controller.
type InferenceGateway interface {
	PredictHourlyWeather(ctx context.Context, req *protoinferencev1.PredictHourlyWeatherRequest) (*protoinferencev1.PredictHourlyWeatherResponse, error)
	PredictDailyWeather(ctx context.Context, req *protoinferencev1.PredictDailyWeatherRequest) (*protoinferencev1.PredictDailyWeatherResponse, error)
}

// Controller coordinates fetching history, generating forecasts, and storing predictions.
type Controller struct {
	log        *zap.Logger
	dbGateway  DatabaseGateway
	infGateway InferenceGateway
	nowFunc    func() time.Time
}

// ControllerParams defines dependencies for Controller.
type ControllerParams struct {
	fx.In

	Logger           *zap.Logger
	DatabaseGateway  *database.Gateway        `optional:"true"`
	InferenceGateway *inferenceserver.Gateway `optional:"true"`
	DBGateway        DatabaseGateway          `optional:"true"`
	InfGateway       InferenceGateway         `optional:"true"`
}

// New creates a new inference Controller instance.
func New(params ControllerParams) (*Controller, error) {
	var dbGateway DatabaseGateway
	if params.DBGateway != nil {
		dbGateway = params.DBGateway
	} else if params.DatabaseGateway != nil {
		dbGateway = params.DatabaseGateway
	}

	var infGateway InferenceGateway
	if params.InfGateway != nil {
		infGateway = params.InfGateway
	} else if params.InferenceGateway != nil {
		infGateway = params.InferenceGateway
	}

	if dbGateway == nil || infGateway == nil {
		return nil, ErrNoGateways
	}

	return &Controller{
		log:        params.Logger,
		dbGateway:  dbGateway,
		infGateway: infGateway,
		nowFunc:    time.Now,
	}, nil
}

// RunHourlyForecast generates and stores hourly forecasts for all nodes.
func (c *Controller) RunHourlyForecast(ctx context.Context, horizonHours int32) error {
	nodes, err := c.dbGateway.ListNodeCoordinates(ctx)
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}

	var hasErrors bool
	for _, node := range nodes {
		nodeID := node.GetNodeId()
		if nodeID == "" {
			continue
		}

		if err := c.RunHourlyForecastForNodeWithCoord(ctx, node, horizonHours); err != nil {
			c.log.Error("failed hourly forecast for node", zap.String("node_id", nodeID), zap.Error(err))
			hasErrors = true
		}
	}

	if hasErrors {
		return errors.New("one or more nodes encountered hourly forecast errors")
	}

	return nil
}

// RunHourlyForecastForNode aggregates history, dispatches hourly inference, and stores predictions for a node.
func (c *Controller) RunHourlyForecastForNode(ctx context.Context, nodeID string, horizonHours int32) error {
	return c.RunHourlyForecastForNodeWithCoord(ctx, &protodatabasev1.NodeCoordinates{NodeId: nodeID}, horizonHours)
}

// RunHourlyForecastForNodeWithCoord runs an hourly forecast for a node taking into account coordinates.
func (c *Controller) RunHourlyForecastForNodeWithCoord(ctx context.Context, node *protodatabasev1.NodeCoordinates, horizonHours int32) error {
	if horizonHours <= 0 {
		return ErrInvalidHorizon
	}

	nodeID := node.GetNodeId()
	historyDuration := 40 * 24 * time.Hour
	bucketSeconds := uint32(3600)

	issueTime := c.nowFunc().UTC().Truncate(time.Hour)
	startTime := issueTime.Add(-historyDuration)

	aggregates, err := c.dbGateway.GetClimateAggregates(ctx, nodeID, startTime, issueTime, bucketSeconds)
	if err != nil {
		return fmt.Errorf("query climate aggregates for node %q: %w", nodeID, err)
	}

	var completedAggregates []*protodatabasev1.ClimateAggregate
	for _, agg := range aggregates {
		if agg.GetBucketStart().AsTime().Before(issueTime) {
			completedAggregates = append(completedAggregates, agg)
		}
	}

	if len(completedAggregates) == 0 {
		c.log.Info("no historical aggregates found for node, skipping", zap.String("node_id", nodeID))
		return nil
	}

	observations := ToTimeSeriesObservations(completedAggregates)

	req := &protoinferencev1.PredictHourlyWeatherRequest{
		NodeId:       nodeID,
		IssueTime:    timestamppb.New(issueTime),
		HorizonHours: horizonHours,
		History:      observations,
		Latitude:     &node.Latitude,
	}

	res, err := c.infGateway.PredictHourlyWeather(ctx, req)
	if err != nil {
		return fmt.Errorf("predict hourly weather for node %q: %w", nodeID, err)
	}

	hourlyPredictions := res.GetPredictions()
	if len(hourlyPredictions) > 0 {
		predictions := make([]*protodatabasev1.Prediction, len(hourlyPredictions))
		for i, p := range hourlyPredictions {
			predictions[i] = &protodatabasev1.Prediction{
				NodeId:             p.GetNodeId(),
				IssueTime:          p.GetIssueTime(),
				HorizonHours:       p.GetHorizonHours(),
				TemperatureC:       p.TemperatureC,
				HumidityPct:        p.HumidityPct,
				RainfallMm:         p.RainfallMm,
				RainfallConfidence: p.RainfallConfidence,
			}
		}
		if err := c.dbGateway.StorePredictions(ctx, predictions); err != nil {
			return fmt.Errorf("store hourly predictions for node %q: %w", nodeID, err)
		}
	}

	c.log.Info("hourly forecast successfully processed",
		zap.String("node_id", nodeID),
		zap.Int32("horizon_hours", horizonHours),
		zap.Int("predictions", len(hourlyPredictions)),
	)

	return nil
}

// RunDailyForecast generates and stores daily forecasts for all nodes.
func (c *Controller) RunDailyForecast(ctx context.Context, horizonDays int32) error {
	nodes, err := c.dbGateway.ListNodeCoordinates(ctx)
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}

	var hasErrors bool
	for _, node := range nodes {
		nodeID := node.GetNodeId()
		if nodeID == "" {
			continue
		}

		if err := c.RunDailyForecastForNodeWithCoord(ctx, node, horizonDays); err != nil {
			c.log.Error("failed daily forecast for node", zap.String("node_id", nodeID), zap.Error(err))
			hasErrors = true
		}
	}

	if hasErrors {
		return errors.New("one or more nodes encountered daily forecast errors")
	}

	return nil
}

// RunDailyForecastForNode aggregates history, dispatches daily inference, and stores predictions for a node.
func (c *Controller) RunDailyForecastForNode(ctx context.Context, nodeID string, horizonDays int32) error {
	return c.RunDailyForecastForNodeWithCoord(ctx, &protodatabasev1.NodeCoordinates{NodeId: nodeID}, horizonDays)
}

// RunDailyForecastForNodeWithCoord runs a daily forecast for a node taking into account coordinates.
func (c *Controller) RunDailyForecastForNodeWithCoord(ctx context.Context, node *protodatabasev1.NodeCoordinates, horizonDays int32) error {
	if horizonDays <= 0 {
		return ErrInvalidHorizon
	}

	nodeID := node.GetNodeId()
	historyDuration := 365 * 24 * time.Hour
	bucketSeconds := uint32(86400)

	now := c.nowFunc()
	startTime := now.Add(-historyDuration)

	aggregates, err := c.dbGateway.GetClimateAggregates(ctx, nodeID, startTime, now, bucketSeconds)
	if err != nil {
		return fmt.Errorf("query climate aggregates for node %q: %w", nodeID, err)
	}

	if len(aggregates) == 0 {
		c.log.Info("no historical aggregates found for node, skipping", zap.String("node_id", nodeID))
		return nil
	}

	observations := ToTimeSeriesObservations(aggregates)

	req := &protoinferencev1.PredictDailyWeatherRequest{
		NodeId:      nodeID,
		IssueTime:   timestamppb.New(now),
		HorizonDays: horizonDays,
		History:     observations,
		Latitude:    &node.Latitude,
	}

	res, err := c.infGateway.PredictDailyWeather(ctx, req)
	if err != nil {
		return fmt.Errorf("predict daily weather for node %q: %w", nodeID, err)
	}

	dailyPredictions := res.GetPredictions()
	if len(dailyPredictions) > 0 {
		predictions := make([]*protodatabasev1.Prediction, len(dailyPredictions))
		for i, p := range dailyPredictions {
			predictions[i] = &protodatabasev1.Prediction{
				NodeId:             p.GetNodeId(),
				IssueTime:          p.GetIssueTime(),
				HorizonHours:       p.GetHorizonDays() * 24,
				MinTemperatureC:    p.MinTemperatureC,
				MaxTemperatureC:    p.MaxTemperatureC,
				MinHumidityPct:     p.MinHumidityPct,
				MaxHumidityPct:     p.MaxHumidityPct,
				RainfallMm:         p.RainfallMm,
				RainfallConfidence: p.RainfallConfidence,
			}
		}
		if err := c.dbGateway.StorePredictions(ctx, predictions); err != nil {
			return fmt.Errorf("store daily predictions for node %q: %w", nodeID, err)
		}
	}

	c.log.Info("daily forecast successfully processed",
		zap.String("node_id", nodeID),
		zap.Int32("horizon_days", horizonDays),
		zap.Int("predictions", len(dailyPredictions)),
	)

	return nil
}
