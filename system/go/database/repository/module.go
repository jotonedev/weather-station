package repository

import (
	"weather-station/database/repository/timestorage"

	"go.uber.org/fx"
)

var Module = fx.Module("repository",
	fx.Provide(
		timestorage.New,
	),
)
