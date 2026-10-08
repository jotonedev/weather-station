package repository

import (
	"go.uber.org/fx"
	"weather-station/database/repository/timestorage"
)

var Module = fx.Module("repository",
	fx.Provide(
		timestorage.New,
	),
)
