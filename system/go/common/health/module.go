package health

import (
	"weather-station/common/fxgroup"

	"go.uber.org/fx"
)

// Module registers the health check handler with the shared servers:
// the HTTP health route and the gRPC health service.
var Module = fx.Module("health",
	fx.Provide(
		fxgroup.AsHTTPRoute(New),
		fxgroup.AsGRPCHandler(NewGRPC),
	),
)
