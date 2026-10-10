package gateway

import (
	"weather-station/inference-job/gateway/database"
	"weather-station/inference-job/gateway/inferenceserver"

	"go.uber.org/fx"
)

// Module wires service-specific gateways (outbound dependencies).
var Module = fx.Module("gateway",
	fx.Provide(
		database.NewConfig,
		database.New,
		inferenceserver.NewConfig,
		inferenceserver.New,
	),
)
