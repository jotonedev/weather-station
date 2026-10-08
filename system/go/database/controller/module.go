package controller

import (
	"weather-station/database/controller/predictiondata"
	"weather-station/database/controller/weatherdata"

	"go.uber.org/fx"
)

// Module wires service-specific controllers.
var Module = fx.Module("controller",
	fx.Provide(
		predictiondata.New,
		weatherdata.New,
	),
)
