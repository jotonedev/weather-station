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

	readings, err := c.db.GetLatestReadings(ctx, selectedNodeID, 1)
	if err != nil {
		c.log.Warn("Failed to fetch latest readings", zap.String("node_id", selectedNodeID), zap.Error(err))
		data.ErrorMessage = "Unable to connect to database service to retrieve latest readings."
		return data
	}

	if len(readings) > 0 {
		latest := pbReadingToView(readings[0])
		data.LatestReading = &latest
	}

	predictions, err := c.db.GetForecast(ctx, selectedNodeID)
	if err != nil {
		c.log.Warn("Failed to fetch forecast predictions", zap.String("node_id", selectedNodeID), zap.Error(err))
		return data
	}

	if len(predictions) > 0 {
		forecast := make([]ForecastItemView, 0, len(predictions))
		hourly := make([]ForecastItemView, 0)
		daily := make([]ForecastItemView, 0)
		for _, pred := range predictions {
			if pred == nil {
				continue
			}
			if data.ForecastIssuedAt == "" && pred.IssueTime != nil {
				issueTime := pred.IssueTime.AsTime()
				if !issueTime.IsZero() {
					data.ForecastIssuedAt = issueTime.UTC().Format("2006-01-02 15:04 UTC")
				}
			}
			view := pbPredictionToView(pred)
			forecast = append(forecast, view)
			if pred.HorizonHours < 24 {
				hourly = append(hourly, view)
			} else {
				daily = append(daily, view)
			}
		}
		data.Forecast = forecast
		data.HourlyForecast = hourly
		data.DailyForecast = daily
	}

	return data
}
