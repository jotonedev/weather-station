package weatherdata

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"weather-station/frontend/gateway/database"
)

// Controller retrieves weather data from the database gateway.
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

// New creates a new weatherdata Controller instance.
func New(params ControllerParams) (*Controller, error) {
	return &Controller{
		log: params.Logger,
		db:  params.Database,
	}, nil
}

// GetDashboardData queries the database gateway and prepares weather data for the dashboard.
func (c *Controller) GetDashboardData(ctx context.Context, requestedNodeID string) DashboardData {
	data := DashboardData{
		Title: "Weather Station Dashboard",
	}

	nodes, err := c.db.ListNodeCoordinates(ctx)
	if err != nil {
		c.log.Warn("Failed to list nodes from database", zap.Error(err))
	}

	var availableNodeIDs []string
	for _, node := range nodes {
		if node != nil && node.NodeId != "" {
			availableNodeIDs = append(availableNodeIDs, node.NodeId)
		}
	}
	data.AvailableNodes = availableNodeIDs

	selectedNodeID := requestedNodeID
	if selectedNodeID == "" && len(availableNodeIDs) > 0 {
		selectedNodeID = availableNodeIDs[0]
	}
	data.NoNodes = len(availableNodeIDs) == 0
	data.SelectedNodeID = selectedNodeID

	if selectedNodeID == "" {
		// No station is registered yet, so there is no data to query.
		return data
	}

	readings, err := c.db.GetLatestReadings(ctx, selectedNodeID, 10)
	if err != nil {
		c.log.Warn("Failed to fetch latest readings", zap.String("node_id", selectedNodeID), zap.Error(err))
		data.ErrorMessage = "Unable to connect to database service to retrieve latest readings."
		return data
	}

	if len(readings) == 0 {
		return data
	}

	latest := pbReadingToView(readings[0])
	data.LatestReading = &latest

	recent := make([]ReadingView, 0, len(readings))
	for _, reading := range readings {
		recent = append(recent, pbReadingToView(reading))
	}
	data.RecentReadings = recent

	return data
}
