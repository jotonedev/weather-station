package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	grpcclient "weather-station/common/client/grpc"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var (
	// ErrNoClient is returned when neither clients nor client factory is provided.
	ErrNoClient = errors.New("client or client factory is required")
	// ErrEmptyNodeID is returned when node ID is empty.
	ErrEmptyNodeID = errors.New("node ID is empty")
)

// Gateway provides access to the database gRPC services for aggregates, nodes, and predictions.
type Gateway struct {
	log              *zap.Logger
	aggregateClient  protodatabasev1.ReadingAggregateServiceClient
	predictionClient protodatabasev1.PredictionDatabaseServiceClient
	nodeClient       protodatabasev1.NodeDatabaseServiceClient
	healthClient     grpc_health_v1.HealthClient
}

// GatewayParams defines dependencies for Gateway.
type GatewayParams struct {
	fx.In

	Logger           *zap.Logger
	Config           Config
	ClientFactory    *grpcclient.ClientFactory                        `optional:"true"`
	AggregateClient  protodatabasev1.ReadingAggregateServiceClient    `optional:"true"`
	PredictionClient protodatabasev1.PredictionDatabaseServiceClient   `optional:"true"`
	NodeClient       protodatabasev1.NodeDatabaseServiceClient        `optional:"true"`
	HealthClient     grpc_health_v1.HealthClient                      `optional:"true"`
}

// New creates a new database Gateway instance.
func New(params GatewayParams) (*Gateway, error) {
	if params.AggregateClient != nil && params.PredictionClient != nil && params.NodeClient != nil {
		return &Gateway{
			log:              params.Logger,
			aggregateClient:  params.AggregateClient,
			predictionClient: params.PredictionClient,
			nodeClient:       params.NodeClient,
			healthClient:     params.HealthClient,
		}, nil
	}

	if params.ClientFactory == nil {
		return nil, ErrNoClient
	}

	conn, err := params.ClientFactory.NewConnection(params.Config.Address)
	if err != nil {
		return nil, fmt.Errorf("connect to database %q: %w", params.Config.Address, err)
	}

	aggregateClient := params.AggregateClient
	if aggregateClient == nil {
		aggregateClient = protodatabasev1.NewReadingAggregateServiceClient(conn)
	}

	predictionClient := params.PredictionClient
	if predictionClient == nil {
		predictionClient = protodatabasev1.NewPredictionDatabaseServiceClient(conn)
	}

	nodeClient := params.NodeClient
	if nodeClient == nil {
		nodeClient = protodatabasev1.NewNodeDatabaseServiceClient(conn)
	}

	return &Gateway{
		log:              params.Logger,
		aggregateClient:  aggregateClient,
		predictionClient: predictionClient,
		nodeClient:       nodeClient,
		healthClient:     grpc_health_v1.NewHealthClient(conn),
	}, nil
}

// Name returns the identifier of the readiness check.
func (*Gateway) Name() string {
	return "database"
}

// Check verifies that the database service is ready.
func (g *Gateway) Check(ctx context.Context) error {
	if g.healthClient == nil {
		return nil
	}
	resp, err := g.healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: "ready"})
	if err != nil {
		return fmt.Errorf("check database health: %w", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("database status is %s", resp.Status)
	}
	return nil
}

// ListNodeCoordinates returns the metadata and identifiers of all configured weather nodes.
func (g *Gateway) ListNodeCoordinates(ctx context.Context) ([]*protodatabasev1.NodeCoordinates, error) {
	res, err := g.nodeClient.ListNodeCoordinates(ctx, &protodatabasev1.ListNodeCoordinatesRequest{})
	if err != nil {
		return nil, fmt.Errorf("list node coordinates: %w", err)
	}

	return res.GetNodes(), nil
}

// GetClimateAggregates retrieves aggregated climate observations for a node within a time range.
func (g *Gateway) GetClimateAggregates(ctx context.Context, nodeID string, start, end time.Time, bucketSeconds uint32) ([]*protodatabasev1.ClimateAggregate, error) {
	if nodeID == "" {
		return nil, ErrEmptyNodeID
	}

	res, err := g.aggregateClient.GetClimateAggregates(ctx, &protodatabasev1.GetClimateAggregatesRequest{
		NodeId:        nodeID,
		StartTime:     timestamppb.New(start),
		EndTime:       timestamppb.New(end),
		BucketSeconds: bucketSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("get climate aggregates for node %q: %w", nodeID, err)
	}

	return res.GetAggregates(), nil
}

// StorePredictions sends a batch of forecast predictions to the database.
func (g *Gateway) StorePredictions(ctx context.Context, predictions []*protodatabasev1.Prediction) error {
	if len(predictions) == 0 {
		return nil
	}

	_, err := g.predictionClient.StorePredictions(ctx, &protodatabasev1.StorePredictionsRequest{
		Predictions: predictions,
	})
	if err != nil {
		return fmt.Errorf("store predictions: %w", err)
	}

	return nil
}
