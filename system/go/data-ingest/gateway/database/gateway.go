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
	// ErrNilReading is returned when attempting to store a nil reading.
	ErrNilReading = errors.New("reading is nil")
	// ErrNoClient is returned when neither a client nor client factory is provided.
	ErrNoClient = errors.New("client or client factory is required")
)

// Gateway provides access to the database gRPC service.
type Gateway struct {
	log    *zap.Logger
	client protodatabasev1.ReadingDatabaseServiceClient
}

// GatewayParams defines dependencies for Gateway.
type GatewayParams struct {
	fx.In

	Logger        *zap.Logger
	Config        Config
	ClientFactory *grpcclient.ClientFactory                    `optional:"true"`
	Client        protodatabasev1.ReadingDatabaseServiceClient `optional:"true"`
}

// New creates a new database Gateway instance.
func New(params GatewayParams) (*Gateway, error) {
	if params.Client != nil {
		return &Gateway{
			log:    params.Logger,
			client: params.Client,
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
		log:    params.Logger,
		client: protodatabasev1.NewReadingDatabaseServiceClient(conn),
	}, nil
}

// StoreReading sends a single reading to the database service.
func (g *Gateway) StoreReading(ctx context.Context, reading *protodatabasev1.Reading) error {
	if reading == nil {
		return ErrNilReading
	}
	return g.StoreReadings(ctx, []*protodatabasev1.Reading{reading})
}

// StoreReadings sends a batch of readings to the database service.
func (g *Gateway) StoreReadings(ctx context.Context, readings []*protodatabasev1.Reading) error {
	if len(readings) == 0 {
		return nil
	}

	_, err := g.client.StoreReadings(ctx, &protodatabasev1.StoreReadingsRequest{
		Readings: readings,
	})
	if err != nil {
		return fmt.Errorf("store readings: %w", err)
	}

	return nil
}
