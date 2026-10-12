package gateway

import (
	"weather-station/common/health"

	"go.uber.org/fx"
)

// Module wires service-specific gateways and readiness checkers.
var Module = fx.Module("gateway",
	fx.Provide(
		health.AsChecker(NewQuestDBChecker),
	),
)
