package client

import (
	"go.uber.org/fx"
	"weather-station/common/client/grpc"
	"weather-station/common/client/http"
	"weather-station/common/client/questdb"
)

var Module = fx.Module("client",
	http.Module,
	grpc.Module,
	questdb.Module,
)
