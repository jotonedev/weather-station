package generator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"weather-station/test-generator/gateway/ingest"
)

var (
	// ErrNoGateway is returned when an IngestGateway dependency is not provided.
	ErrNoGateway = errors.New("ingest gateway is required")
)

// IngestGateway abstracts sending generated readings to data-ingest.
type IngestGateway interface {
	SendReadings(ctx context.Context, readings []ingest.ReadingPayload) error
}

// Controller generates realistic weather readings and dispatches them to the ingest gateway.
type Controller struct {
	log       *zap.Logger
	cfg       Config
	gateway   IngestGateway
	simulator *Simulator
	nowFunc   func() time.Time
}

// ControllerParams defines dependencies for Controller.
type ControllerParams struct {
	fx.In

	Logger        *zap.Logger
	Config        Config
	Gateway       *ingest.Gateway `optional:"true"`
	CustomGateway IngestGateway   `optional:"true"`
}

// New creates a new generator Controller instance.
func New(params ControllerParams) (*Controller, error) {
	var gw IngestGateway
	if params.CustomGateway != nil {
		gw = params.CustomGateway
	} else if params.Gateway != nil {
		gw = params.Gateway
	}
	if gw == nil {
		return nil, ErrNoGateway
	}

	return &Controller{
		log:       params.Logger,
		cfg:       params.Config,
		gateway:   gw,
		simulator: NewSimulator(),
		nowFunc:   time.Now,
	}, nil
}

// Interval returns the configured generation interval.
func (c *Controller) Interval() time.Duration {
	if c.cfg.Interval <= 0 {
		return defaultInterval
	}
	return c.cfg.Interval
}

// GenerateAndSend generates one reading per configured node and sends the batch to data-ingest.
func (c *Controller) GenerateAndSend(ctx context.Context) error {
	now := c.nowFunc().UTC()
	readings := make([]ingest.ReadingPayload, 0, len(c.cfg.NodeIDs))
	for _, nodeID := range c.cfg.NodeIDs {
		if nodeID == "" {
			continue
		}
		readings = append(readings, c.simulator.NextReading(nodeID, now))
	}

	if len(readings) == 0 {
		return nil
	}

	if err := c.gateway.SendReadings(ctx, readings); err != nil {
		return fmt.Errorf("send generated readings: %w", err)
	}

	c.log.Debug("sent generated weather readings", zap.Int("count", len(readings)))
	return nil
}
