package gateway

import (
	"weather-station/common/health"
	"weather-station/inference-job/gateway/database"
	"weather-station/inference-job/gateway/inferenceserver"

	"go.uber.org/fx"
)

// Module wires service-specific gateways and readiness checkers.
var Module = fx.Module("gateway",
	fx.Provide(
		database.NewConfig,
		database.New,
		inferenceserver.NewConfig,
		inferenceserver.New,
		health.AsChecker(func(g *database.Gateway) health.Checker { return g }),
		health.AsChecker(func(g *inferenceserver.Gateway) health.Checker { return g }),
	),
)
