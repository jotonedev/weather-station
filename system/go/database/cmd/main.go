package main

import (
	"weather-station/common/health"
	"weather-station/common/logging"
	"weather-station/common/server"

	dbclient "weather-station/database/client"
	"weather-station/database/controller"
	"weather-station/database/handler"
	"weather-station/database/repository"

	"go.uber.org/fx"
)

func main() {
	fx.New(
		// Shared ingress modules
		server.GRPCModule,
		// Shared logging and health checks (gRPC)
		logging.Module,
		health.Module,
		// Database-specific modules
		dbclient.Module,
		handler.Module,
		controller.Module,
		repository.Module,
	).Run()
}
