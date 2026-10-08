package controller

import (
	"go.uber.org/fx"

	weatherstation "weather-station/data-injest/controller/weatherstation"
)

// Module wires service-specific controllers.
var Module = fx.Module("controller",
	fx.Provide(
		weatherstation.New,
	),
)
