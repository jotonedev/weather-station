package repository

import (
	"go.uber.org/fx"
	"weather-station/database/repository/questdb"
)

// Module aggregates the database storage implementations.
// SQL lives in submodules (e.g. questdb); transport comes from
// weather-station/common/client/questdb via client.Module.
var Module = fx.Module("repository",
	fx.Provide(
		questdb.New,
	),
)
