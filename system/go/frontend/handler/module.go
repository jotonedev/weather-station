package handler

import (
	"weather-station/common/fxgroup"
	"weather-station/frontend/handler/website"

	"go.uber.org/fx"
)

// Module registers HTTP routes for the frontend service.
var Module = fx.Module("handler",
	fx.Provide(
		fxgroup.AsHTTPRoute(website.New),
	),
)
