package client

import (
	"weather-station/common/health"

	"go.uber.org/fx"
)

// Module provides the DuckDB Client and registers it with the health checking system.
var Module = fx.Module("database-client",
	fx.Provide(
		NewConfig,
		NewClient,
		health.AsChecker(func(c *Client) health.Checker {
			return c
		}),
	),
)
