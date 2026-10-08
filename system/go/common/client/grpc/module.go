package grpc

import (
	"go.uber.org/fx"
)

var Module = fx.Module("grpc-client",
	fx.Provide(
		NewClientFactory,
	),
)
