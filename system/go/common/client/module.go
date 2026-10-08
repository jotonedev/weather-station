package client

import (
	"weather-station/common/client/grpc"
	"weather-station/common/client/http"
	"weather-station/common/client/questdb"

	"go.uber.org/fx"
)

var Module = fx.Module("client",
	http.Module,
	grpc.Module,
	questdb.Module,
)
