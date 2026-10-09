package handler

import (
	"weather-station/common/fxgroup"
	"weather-station/database/handler/aggregate"
	"weather-station/database/handler/nodedata"
	"weather-station/database/handler/predictiondata"
	"weather-station/database/handler/readingdata"

	"go.uber.org/fx"
)

// Module wires service-specific handlers as gRPC services on the shared gRPC server.
var Module = fx.Module("handler",
	fx.Provide(
		fxgroup.AsGRPCHandler(predictiondata.New),
		fxgroup.AsGRPCHandler(readingdata.New),
		fxgroup.AsGRPCHandler(nodedata.New),
		fxgroup.AsGRPCHandler(aggregate.New),
	),
)
