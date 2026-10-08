package weatherstation

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	database "weather-station/data-injest/gateway/database"
)

type Controller struct {
	log *zap.Logger
	db  *database.Gateway
}

type ControllerParams struct {
	fx.In

	Logger   *zap.Logger
	Database *database.Gateway
}

func New(params ControllerParams) (*Controller, error) {
	return &Controller{
		log: params.Logger,
		db:  params.Database,
	}, nil
}
