package gateway

import (
	"weather-station/common/health"
	"weather-station/frontend/gateway/database"

	"go.uber.org/fx"
)

// Module wires service-specific gateways and readiness checkers.
var Module = fx.Module("gateway",
	fx.Provide(
		database.NewConfig,
		database.New,
		health.AsChecker(func(g *database.Gateway) health.Checker { return g }),
	),
)
