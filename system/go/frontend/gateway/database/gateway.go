package database

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/fx"
	"go.uber.org/zap"

	grpcclient "weather-station/common/client/grpc"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var (
	// ErrNoClient is returned when neither clients nor client factory are provided.
	ErrNoClient = errors.New("client or client factory is required")
)

// Gateway provides access to the database gRPC services for the frontend.
type Gateway struct {
	log              *zap.Logger
	readingClient    protodatabasev1.ReadingDatabaseServiceClient
	predictionClient protodatabasev1.PredictionDatabaseServiceClient
	nodeClient       protodatabasev1.NodeDatabaseServiceClient
	healthClient     grpc_health_v1.HealthClient
}

// GatewayParams defines dependencies for Gateway.
type GatewayParams struct {
	fx.In

	Logger           *zap.Logger
	Config           Config
	ClientFactory    *grpcclient.ClientFactory                       `optional:"true"`
	ReadingClient    protodatabasev1.ReadingDatabaseServiceClient    `optional:"true"`
	PredictionClient protodatabasev1.PredictionDatabaseServiceClient `optional:"true"`
	NodeClient       protodatabasev1.NodeDatabaseServiceClient       `optional:"true"`
	HealthClient     grpc_health_v1.HealthClient                     `optional:"true"`
}

// New creates a new database Gateway instance for the frontend.
func New(params GatewayParams) (*Gateway, error) {
	if params.ReadingClient != nil || params.NodeClient != nil || params.PredictionClient != nil {
		return &Gateway{
			log:              params.Logger,
			readingClient:    params.ReadingClient,
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

	return &Gateway{
		log:              params.Logger,
		readingClient:    protodatabasev1.NewReadingDatabaseServiceClient(conn),
		predictionClient: protodatabasev1.NewPredictionDatabaseServiceClient(conn),
		nodeClient:       protodatabasev1.NewNodeDatabaseServiceClient(conn),
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

// GetLatestReadings retrieves the most recent raw readings for a given node.
func (g *Gateway) GetLatestReadings(ctx context.Context, nodeID string, limit uint32) ([]*protodatabasev1.Reading, error) {
	resp, err := g.readingClient.GetLatestReadings(ctx, &protodatabasev1.GetLatestReadingsRequest{
		NodeId: nodeID,
		Limit:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get latest readings for node %q: %w", nodeID, err)
	}
	return resp.Readings, nil
}

// GetForecast retrieves the latest forecast predictions for a given node.
func (g *Gateway) GetForecast(ctx context.Context, nodeID string) ([]*protodatabasev1.Prediction, error) {
	if g.predictionClient == nil {
		return nil, nil
	}
	resp, err := g.predictionClient.GetForecast(ctx, &protodatabasev1.GetForecastRequest{
		NodeId: nodeID,
	})
	if err != nil {
		return nil, fmt.Errorf("get forecast for node %q: %w", nodeID, err)
	}
	return resp.GetPredictions(), nil
}

// ListNodeCoordinates retrieves location coordinates for all registered nodes.
func (g *Gateway) ListNodeCoordinates(ctx context.Context) ([]*protodatabasev1.NodeCoordinates, error) {
	if g.nodeClient == nil {
		return nil, nil
	}
	resp, err := g.nodeClient.ListNodeCoordinates(ctx, &protodatabasev1.ListNodeCoordinatesRequest{})
	if err != nil {
		return nil, fmt.Errorf("list node coordinates: %w", err)
	}
	return resp.Nodes, nil
}

// CreateNode creates or updates a node's coordinates in the database.
func (g *Gateway) CreateNode(ctx context.Context, node *protodatabasev1.NodeCoordinates) error {
	if g.nodeClient == nil {
		return ErrNoClient
	}
	_, err := g.nodeClient.CreateNode(ctx, &protodatabasev1.CreateNodeRequest{
		Node: node,
	})
	if err != nil {
		return fmt.Errorf("create node %q: %w", node.GetNodeId(), err)
	}
	return nil
}

// DeleteNode removes a node and its readings and predictions from the database.
func (g *Gateway) DeleteNode(ctx context.Context, nodeID string) error {
	if g.nodeClient == nil {
		return ErrNoClient
	}
	_, err := g.nodeClient.DeleteNode(ctx, &protodatabasev1.DeleteNodeRequest{
		NodeId: nodeID,
	})
	if err != nil {
		return fmt.Errorf("delete node %q: %w", nodeID, err)
	}
	return nil
}
