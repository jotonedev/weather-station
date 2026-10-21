package controller

import (
	"go.uber.org/fx"

	"weather-station/test-generator/controller/generator"
)

// Module wires controller dependencies and starts the continuous test data generator.
var Module = fx.Module("controller",
	fx.Provide(
		generator.NewConfig,
		generator.New,
	),
	fx.Invoke(RunGenerator),
)
