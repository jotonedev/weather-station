package main

import (
	"weather-station/common/client"
	"weather-station/common/health"
	"weather-station/common/logging"
	"weather-station/common/server"

	"weather-station/database/controller"
	"weather-station/database/gateway"
	"weather-station/database/handler"
	"weather-station/database/repository"

	"go.uber.org/fx"
)

func main() {
	fx.New(
		// Shared ingress / egress modules
		server.Module,
		client.Module,
		// Shared logging and health checks (HTTP + gRPC)
		logging.Module,
		health.Module,
		// Service-specific modules
		handler.Module,
		gateway.Module,
		controller.Module,
		repository.Module,
	).Run()
}
