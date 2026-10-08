package handler

import (
	"go.uber.org/fx"
)

// Module wires service-specific handlers.
// Register HTTP routes and gRPC handlers here using the shared helpers:
//
//	fx.Provide(fxgroup.AsHTTPRoute(NewMyHandler))
//	fx.Provide(fxgroup.AsGRPCHandler(NewMyGRPC))
var Module = fx.Module("handler",
	fx.Provide(),
)
