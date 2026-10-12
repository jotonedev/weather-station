package controller

import (
	"weather-station/inference-job/controller/inference"

	"go.uber.org/fx"
)

// Module wires service-specific controllers and triggers job execution.
var Module = fx.Module("controller",
	fx.Provide(
		inference.New,
	),
	fx.Invoke(
		RunInferenceJob,
	),
)
