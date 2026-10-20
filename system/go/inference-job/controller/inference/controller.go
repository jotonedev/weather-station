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
	// ErrInvalidMode is returned when an unsupported prediction mode is supplied.
	ErrInvalidMode = errors.New("invalid prediction mode")
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
	PredictWeather(ctx context.Context, req *protoinferencev1.PredictWeatherRequest) (*protoinferencev1.PredictWeatherResponse, error)
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

// RunForecast generates and stores forecasts for all nodes in the given mode.
func (c *Controller) RunForecast(ctx context.Context, mode protoinferencev1.PredictionMode) error {
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

		if err := c.RunForecastForNode(ctx, nodeID, mode); err != nil {
			c.log.Error("failed forecast for node", zap.String("node_id", nodeID), zap.Error(err))
			hasErrors = true
		}
	}

	if hasErrors {
		return errors.New("one or more nodes encountered forecast errors")
	}

	return nil
}

// RunForecastForNode aggregates history, dispatches inference, and stores predictions for a single node.
func (c *Controller) RunForecastForNode(ctx context.Context, nodeID string, mode protoinferencev1.PredictionMode) error {
	var historyDuration time.Duration
	var bucketSeconds uint32

	switch mode {
	case protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_24_HOURS:
		// Up to last 40 days aggregated hourly.
		historyDuration = 40 * 24 * time.Hour
		bucketSeconds = 3600
	case protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_10_DAYS:
		// Up to last 1 year aggregated daily.
		historyDuration = 365 * 24 * time.Hour
		bucketSeconds = 86400
	default:
		return ErrInvalidMode
	}

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

	req := &protoinferencev1.PredictWeatherRequest{
		NodeId:    nodeID,
		Mode:      mode,
		IssueTime: timestamppb.New(now),
		History:   observations,
	}

	res, err := c.infGateway.PredictWeather(ctx, req)
	if err != nil {
		return fmt.Errorf("predict weather for node %q: %w", nodeID, err)
	}

	predictions := res.GetPredictions()
	if len(predictions) > 0 {
		if err := c.dbGateway.StorePredictions(ctx, predictions); err != nil {
			return fmt.Errorf("store predictions for node %q: %w", nodeID, err)
		}
	}

	c.log.Info("forecast successfully processed",
		zap.String("node_id", nodeID),
		zap.String("mode", mode.String()),
		zap.Int("predictions", len(predictions)),
	)

	return nil
}
