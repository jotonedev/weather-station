package controller

import (
	"go.uber.org/fx"

	homepage "weather-station/frontend/controller/homepage"
	weatherdata "weather-station/frontend/controller/weatherdata"
)

// Module wires service-specific controllers.
var Module = fx.Module("controller",
	fx.Provide(
		weatherdata.New,
		homepage.New,
	),
)
