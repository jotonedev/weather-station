package database

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/fx"
	"go.uber.org/zap"

	grpcclient "weather-station/common/client/grpc"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

var (
	// ErrNoClient is returned when neither clients nor client factory are provided.
	ErrNoClient = errors.New("client or client factory is required")
)

// Gateway provides access to the database gRPC services for the frontend.
type Gateway struct {
	log           *zap.Logger
	readingClient protodatabasev1.ReadingDatabaseServiceClient
	nodeClient    protodatabasev1.NodeDatabaseServiceClient
}

// GatewayParams defines dependencies for Gateway.
type GatewayParams struct {
	fx.In

	Logger        *zap.Logger
	Config        Config
	ClientFactory *grpcclient.ClientFactory                    `optional:"true"`
	ReadingClient protodatabasev1.ReadingDatabaseServiceClient `optional:"true"`
	NodeClient    protodatabasev1.NodeDatabaseServiceClient    `optional:"true"`
}

// New creates a new database Gateway instance for the frontend.
func New(params GatewayParams) (*Gateway, error) {
	if params.ReadingClient != nil {
		return &Gateway{
			log:           params.Logger,
			readingClient: params.ReadingClient,
			nodeClient:    params.NodeClient,
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
		log:           params.Logger,
		readingClient: protodatabasev1.NewReadingDatabaseServiceClient(conn),
		nodeClient:    protodatabasev1.NewNodeDatabaseServiceClient(conn),
	}, nil
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
