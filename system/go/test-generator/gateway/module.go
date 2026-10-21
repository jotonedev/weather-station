package gateway

import (
	"weather-station/test-generator/gateway/ingest"

	"go.uber.org/fx"
)

// Module wires egress gateways for the test-generator service.
var Module = fx.Module("gateway",
	fx.Provide(
		ingest.NewConfig,
		ingest.New,
	),
)
