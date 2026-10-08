package server

import (
	"weather-station/common/server/grpc"
	"weather-station/common/server/http"

	"go.uber.org/fx"
)

var Module = fx.Module("server",
	http.Module,
	grpc.Module,
)
