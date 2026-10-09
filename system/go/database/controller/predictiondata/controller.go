package predictiondata

import (
	"weather-station/database/repository/prediction"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Controller struct {
	log  *zap.Logger
	repo *prediction.Repository
}

type ControllerParams struct {
	fx.In

	Logger *zap.Logger
	Repo   *prediction.Repository
}

func New(params ControllerParams) (*Controller, error) {
	return &Controller{
		log:  params.Logger,
		repo: params.Repo,
	}, nil
}
