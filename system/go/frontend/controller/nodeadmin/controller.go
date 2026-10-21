package nodeadmin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"weather-station/frontend/gateway/database"
)

var (
	// ErrMissingNodeID is returned when node_id is blank.
	ErrMissingNodeID = errors.New("node_id is required")
	// ErrInvalidLatitude is returned when latitude is outside [-90, 90].
	ErrInvalidLatitude = errors.New("latitude must be between -90 and 90")
	// ErrInvalidLongitude is returned when longitude is outside [-180, 180].
	ErrInvalidLongitude = errors.New("longitude must be between -180 and 180")
)

// Controller manages station node administration logic and HTML rendering.
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

// New creates a new nodeadmin Controller instance.
func New(params ControllerParams) (*Controller, error) {
	return &Controller{
		log: params.Logger,
		db:  params.Database,
	}, nil
}

// SaveNode validates form input and stores node coordinates via the database gateway.
func (c *Controller) SaveNode(ctx context.Context, input NodeFormInput) error {
	nodeID := strings.TrimSpace(input.NodeID)
	if nodeID == "" {
		return ErrMissingNodeID
	}
	if input.Latitude < -90 || input.Latitude > 90 {
		return ErrInvalidLatitude
	}
	if input.Longitude < -180 || input.Longitude > 180 {
		return ErrInvalidLongitude
	}

	if err := c.db.CreateNode(ctx, formInputToPB(input)); err != nil {
		return fmt.Errorf("save node %q: %w", nodeID, err)
	}
	return nil
}

// DeleteNode validates the node ID and deletes the node and its data via the database gateway.
func (c *Controller) DeleteNode(ctx context.Context, nodeID string) error {
	trimmedID := strings.TrimSpace(nodeID)
	if trimmedID == "" {
		return ErrMissingNodeID
	}
	if err := c.db.DeleteNode(ctx, trimmedID); err != nil {
		return fmt.Errorf("delete node %q: %w", trimmedID, err)
	}
	return nil
}

// Render queries registered nodes and renders the node management HTML page.
func (c *Controller) Render(ctx context.Context, statusMessage, errorMessage string) ([]byte, error) {
	data := PageData{
		Title:         "Manage Nodes - Weather Station",
		StatusMessage: statusMessage,
		ErrorMessage:  errorMessage,
	}

	nodes, err := c.db.ListNodeCoordinates(ctx)
	if err != nil {
		c.log.Warn("Failed to list nodes for admin view", zap.Error(err))
		if data.ErrorMessage == "" {
			data.ErrorMessage = "Unable to load registered nodes from database."
		}
	}

	views := make([]NodeItemView, 0, len(nodes))
	for _, node := range nodes {
		if node == nil || node.GetNodeId() == "" {
			continue
		}
		views = append(views, pbNodeToView(node))
	}
	data.Nodes = views

	var buffer bytes.Buffer
	if err := nodesTmpl.Execute(&buffer, data); err != nil {
		return nil, fmt.Errorf("execute nodes template: %w", err)
	}
	return buffer.Bytes(), nil
}
