package controller

import (
	"weather-station/database/controller/aggregate"
	"weather-station/database/controller/nodedata"
	"weather-station/database/controller/predictiondata"
	"weather-station/database/controller/readingdata"

	"go.uber.org/fx"
)

// Module wires service-specific controllers.
var Module = fx.Module("controller",
	fx.Provide(
		predictiondata.New,
		readingdata.New,
		nodedata.New,
		aggregate.New,
	),
)
