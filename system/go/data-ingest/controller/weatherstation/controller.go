package weatherstation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"weather-station/data-ingest/gateway/database"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

var (
	// ErrMissingNodeID is returned when a reading lacks a node ID.
	ErrMissingNodeID = errors.New("node_id is required")
)

// Controller handles business logic for incoming weather station readings.
type Controller struct {
	log *zap.Logger
	db  *database.Gateway
}

// ControllerParams defines dependencies for Controller.
type ControllerParams struct {
	fx.In

	Logger   *zap.Logger
	Database *database.Gateway
}

// New creates a new weatherstation Controller instance.
func New(params ControllerParams) (*Controller, error) {
	return &Controller{
		log: params.Logger,
		db:  params.Database,
	}, nil
}

// StoreReading validates and forwards a weather reading to the database gateway.
func (c *Controller) StoreReading(ctx context.Context, reading Reading) error {
	if reading.NodeID == "" {
		return ErrMissingNodeID
	}
	if reading.Timestamp.IsZero() {
		reading.Timestamp = time.Now().UTC()
	}

	pbReading := readingToPB(reading)
	if err := c.db.StoreReading(ctx, pbReading); err != nil {
		return fmt.Errorf("store reading for node %q: %w", reading.NodeID, err)
	}

	return nil
}

// StoreReadings validates and forwards a batch of weather readings to the database gateway.
func (c *Controller) StoreReadings(ctx context.Context, readings []Reading) error {
	if len(readings) == 0 {
		return nil
	}

	pbReadings := make([]*protodatabasev1.Reading, 0, len(readings))
	now := time.Now().UTC()
	for index, reading := range readings {
		if reading.NodeID == "" {
			return fmt.Errorf("reading at index %d: %w", index, ErrMissingNodeID)
		}
		if reading.Timestamp.IsZero() {
			reading.Timestamp = now
		}
		pbReadings = append(pbReadings, readingToPB(reading))
	}

	if err := c.db.StoreReadings(ctx, pbReadings); err != nil {
		return fmt.Errorf("store readings batch: %w", err)
	}

	return nil
}
