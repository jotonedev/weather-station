package health

import (
	"weather-station/common/fxgroup"

	"go.uber.org/fx"
)

// Module registers health and readiness check handlers with shared servers:
// the HTTP health route, the HTTP ready route, and the gRPC health service.
var Module = fx.Module("health",
	fx.Provide(
		fxgroup.AsHTTPRoute(NewHealth),
		fxgroup.AsHTTPRoute(NewReady),
		fxgroup.AsGRPCHandler(NewGRPC),
	),
)
