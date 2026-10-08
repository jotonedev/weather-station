package weatherdata

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
	"weather-station/database/repository/timestorage"
)

type Controller struct {
	log *zap.Logger
	repo *timestorage.Repository
}

type ControllerParams struct {
	fx.In
	
	Logger *zap.Logger
}

func New(params ControllerParams) (*Controller, error) {
	return &Controller{
		log: params.Logger,
	}, nil
}

