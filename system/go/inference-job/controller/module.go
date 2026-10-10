package controller

import (
	"weather-station/inference-job/controller/inference"

	"go.uber.org/fx"
)

// Module wires service-specific controllers.
var Module = fx.Module("controller",
	fx.Provide(
		inference.New,
	),
)
