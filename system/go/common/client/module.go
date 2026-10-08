package client

import (
	"go.uber.org/fx"
	"weather-station/common/client/grpc"
	"weather-station/common/client/http"
)

var Module = fx.Module("client",
	http.Module,
	grpc.Module,
)
