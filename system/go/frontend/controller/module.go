package controller

import (
	"go.uber.org/fx"

	homepage "weather-station/frontend/controller/homepage"
)

// Module wires service-specific controllers.
var Module = fx.Module("controller",
	fx.Provide(
		homepage.New,
	),
)
