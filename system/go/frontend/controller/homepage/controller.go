package homepage

import (
	"bytes"
	"context"
	"fmt"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"weather-station/frontend/controller/weatherdata"
)

// Controller handles HTML template rendering for the homepage.
type Controller struct {
	log         *zap.Logger
	weatherData *weatherdata.Controller
}

// ControllerParams defines dependencies for Controller.
type ControllerParams struct {
	fx.In

	Logger      *zap.Logger
	WeatherData *weatherdata.Controller
}

// New creates a new homepage template Controller instance.
func New(params ControllerParams) (*Controller, error) {
	return &Controller{
		log:         params.Logger,
		weatherData: params.WeatherData,
	}, nil
}

// Render retrieves data from the data controller and generates the HTML dashboard page.
func (c *Controller) Render(ctx context.Context, requestedNodeID string) ([]byte, error) {
	data := c.weatherData.GetDashboardData(ctx, requestedNodeID)

	var buffer bytes.Buffer
	if err := indexTmpl.Execute(&buffer, data); err != nil {
		return nil, fmt.Errorf("execute homepage template: %w", err)
	}

	return buffer.Bytes(), nil
}
