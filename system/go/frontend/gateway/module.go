package gateway

import (
	"go.uber.org/fx"

	"weather-station/frontend/gateway/database"
)

// Module wires service-specific gateways (outbound dependencies).
var Module = fx.Module("gateway",
	fx.Provide(
		database.NewConfig,
		database.New,
	),
)
