package grpc

import (
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

var Module = fx.Module("grpc",
	fx.Provide(
		fx.Annotate(
			NewGRPCServer,
			fx.ParamTags("", "", `group:"grpc-handlers"`),
		),
	),
	fx.Invoke(func(*grpc.Server) {}),
)
