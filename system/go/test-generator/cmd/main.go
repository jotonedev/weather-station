package main

import (
	"weather-station/common/client"
	"weather-station/common/health"
	"weather-station/common/logging"
	"weather-station/common/server"

	"weather-station/test-generator/controller"
	"weather-station/test-generator/gateway"

	"go.uber.org/fx"
)

func main() {
	fx.New(
		server.GRPCModule,
		client.Module,
		logging.Module,
		health.Module,
		gateway.Module,
		controller.Module,
	).Run()
}
