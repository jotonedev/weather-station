package weatherdata

import (
	"context"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
	"weather-station/frontend/gateway/database"
)

const minForecastConfidencePct = 10

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
		var hourlyPreds []*protodatabasev1.Prediction
		var dailyPreds []*protodatabasev1.Prediction

		for _, pred := range predictions {
			if pred == nil {
				continue
			}
			if pred.HorizonHours < 24 {
				hourlyPreds = append(hourlyPreds, pred)
			} else {
				dailyPreds = append(dailyPreds, pred)
			}
		}

		hourly := make([]ForecastItemView, 0, len(hourlyPreds))
		for _, pred := range hourlyPreds {
			if pred.RainfallConfidence != nil {
				_, pct := formatConfidence(pred.RainfallConfidence)
				if pct < minForecastConfidencePct {
					break
				}
			}
			if pred.IssueTime != nil && !pred.IssueTime.AsTime().IsZero() {
				utcIssue := pred.IssueTime.AsTime().UTC()
				formattedIssue := utcIssue.Format("2006-01-02 15:04 UTC")
				isoIssue := utcIssue.Format(time.RFC3339)
				if data.ForecastIssuedAt == "" {
					data.ForecastIssuedAt = formattedIssue
					data.ForecastIssuedAtISO = isoIssue
				}
				if data.HourlyIssuedAt == "" {
					data.HourlyIssuedAt = formattedIssue
					data.HourlyIssuedAtISO = isoIssue
				}
			}
			hourly = append(hourly, pbPredictionToView(pred))
		}

		daily := make([]ForecastItemView, 0, len(dailyPreds))
		for _, pred := range dailyPreds {
			if pred.RainfallConfidence != nil {
				_, pct := formatConfidence(pred.RainfallConfidence)
				if pct < minForecastConfidencePct {
					break
				}
			}
			if pred.IssueTime != nil && !pred.IssueTime.AsTime().IsZero() {
				utcIssue := pred.IssueTime.AsTime().UTC()
				formattedIssue := utcIssue.Format("2006-01-02 15:04 UTC")
				isoIssue := utcIssue.Format(time.RFC3339)
				if data.ForecastIssuedAt == "" {
					data.ForecastIssuedAt = formattedIssue
					data.ForecastIssuedAtISO = isoIssue
				}
				if data.DailyIssuedAt == "" {
					data.DailyIssuedAt = formattedIssue
					data.DailyIssuedAtISO = isoIssue
				}
			}
			daily = append(daily, pbPredictionToView(pred))
		}

		forecast := make([]ForecastItemView, 0, len(hourly)+len(daily))
		forecast = append(forecast, hourly...)
		forecast = append(forecast, daily...)

		data.Forecast = forecast
		data.HourlyForecast = hourly
		data.DailyForecast = daily
	}

	return data
}
