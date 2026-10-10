package gateway

import (
	"go.uber.org/fx"

	"weather-station/data-ingest/gateway/database"
)

// Module wires service-specific gateways (outbound dependencies).
var Module = fx.Module("gateway",
	fx.Provide(
		database.NewConfig,
		database.New,
	),
)
