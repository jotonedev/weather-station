package server

import (
	"weather-station/common/server/grpc"
	"weather-station/common/server/http"

	"go.uber.org/fx"
)

// GRPCModule registers the shared gRPC server.
var GRPCModule = grpc.Module

// HTTPModule registers the shared HTTP server.
var HTTPModule = http.Module

// Module registers both shared HTTP and gRPC servers.
var Module = fx.Module("server",
	http.Module,
	grpc.Module,
)
