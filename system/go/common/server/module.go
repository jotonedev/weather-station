package server

import (
	"go.uber.org/fx"
	"weather-station/common/server/grpc"
	"weather-station/common/server/http"
)

var Module = fx.Module("server",
	http.Module,
	grpc.Module,
)
