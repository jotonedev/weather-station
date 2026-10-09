package fxgroup

import (
	"weather-station/common/server/grpc"
	"weather-station/common/server/http"

	"go.uber.org/fx"
)

// AsHTTPRoute annotates a provider as an HTTP route so the shared HTTP server
// registers it. The provider must return a value implementing http.Route.
func AsHTTPRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(http.Route)),
		fx.ResultTags(`group:"http-routes"`),
	)
}

// AsGRPCHandler annotates a provider as a gRPC registrar so the shared gRPC
// server registers it. The provider must return a value implementing grpc.Registrar.
func AsGRPCHandler(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(grpc.Registrar)),
		fx.ResultTags(`group:"grpc-handlers"`),
	)
}
