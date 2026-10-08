package gateway

import (
	"go.uber.org/fx"

	database "weather-station/data-injest/gateway/database"
)

// Module wires service-specific gateways (outbound dependencies).
var Module = fx.Module("gateway",
	fx.Provide(
		database.New,
	),
)
