package client

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Client wraps an embedded DuckDB database connection, provides schema management,
// and implements health.Checker for gRPC readiness probes.
type Client struct {
	db  *sql.DB
	log *zap.Logger
}

// NewClient initializes the DuckDB instance, configures PRAGMAs, and executes initial migrations.
func NewClient(lc fx.Lifecycle, cfg *Config, log *zap.Logger) (*Client, error) {
	if cfg.Path != "" && cfg.Path != ":memory:" {
		dir := filepath.Dir(cfg.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create duckdb directory %q: %w", dir, err)
		}
	}

	db, err := sql.Open("duckdb", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb at %q: %w", cfg.Path, err)
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)

	c := &Client{db: db, log: log}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
			defer cancel()

			if err := db.PingContext(pingCtx); err != nil {
				c.log.Error("DuckDB ping failed", zap.Error(err))
				return err
			}

			if cfg.MaxMemory != "" {
				if _, err := db.ExecContext(ctx, fmt.Sprintf("PRAGMA max_memory = '%s';", cfg.MaxMemory)); err != nil {
					c.log.Warn("failed to set duckdb max_memory pragma", zap.Error(err))
				}
			}
			if cfg.Threads > 0 {
				if _, err := db.ExecContext(ctx, fmt.Sprintf("PRAGMA threads = %d;", cfg.Threads)); err != nil {
					c.log.Warn("failed to set duckdb threads pragma", zap.Error(err))
				}
			}

			if err := c.initSchema(ctx); err != nil {
				c.log.Error("DuckDB schema initialization failed", zap.Error(err))
				return fmt.Errorf("init schema: %w", err)
			}

			c.log.Info("DuckDB initialized successfully",
				zap.String("path", cfg.Path),
				zap.String("max_memory", cfg.MaxMemory),
				zap.Int("threads", cfg.Threads),
			)
			return nil
		},
		OnStop: func(_ context.Context) error {
			c.log.Info("Closing DuckDB connection")
			return db.Close()
		},
	})

	return c, nil
}

// DB returns the underlying sql.DB instance.
func (c *Client) DB() *sql.DB {
	return c.db
}

// Ping checks if the database is reachable and ready to service queries.
func (c *Client) Ping(ctx context.Context) error {
	if c.db == nil {
		return fmt.Errorf("database client is nil")
	}
	return c.db.PingContext(ctx)
}

// Name implements health.Checker.
func (c *Client) Name() string {
	return "duckdb"
}

// Check implements health.Checker for readiness checking.
func (c *Client) Check(ctx context.Context) error {
	return c.Ping(ctx)
}

func (c *Client) initSchema(ctx context.Context) error {
	queries := []string{
		// 1. Weather readings table
		`CREATE TABLE IF NOT EXISTS weather (
			node_id            VARCHAR NOT NULL,
			ts                 TIMESTAMP NOT NULL,
			wind_speed         FLOAT,
			wind_angle         FLOAT,
			raw_wind_speed     FLOAT,
			raw_wind_angle     FLOAT,
			temperature        FLOAT,
			humidity           FLOAT,
			pressure           FLOAT,
			rain_1h            FLOAT,
			rain_24h           FLOAT,
			rain_rate          FLOAT,
			pm1_0              INTEGER,
			pm2_5              INTEGER,
			pm10_0             INTEGER,
			voc                INTEGER,
			anemo_status       INTEGER,
			anemo_agc          INTEGER,
			anemo_agc_variance INTEGER,
			vane_status        INTEGER,
			vane_agc           INTEGER,
			vane_agc_variance  INTEGER,
			rain_status        INTEGER,
			rain_agc           INTEGER,
			rain_agc_variance  INTEGER
		);`,
		`CREATE INDEX IF NOT EXISTS idx_weather_node_ts ON weather (node_id, ts);`,

		// 2. Node coordinates metadata table
		`CREATE TABLE IF NOT EXISTS node_coordinates (
			node_id       VARCHAR PRIMARY KEY,
			latitude      DOUBLE NOT NULL,
			longitude     DOUBLE NOT NULL,
			elevation_m   DOUBLE NOT NULL,
			location_name VARCHAR
		);`,

		// 3. Hourly forecast predictions table
		`CREATE TABLE IF NOT EXISTS hourly_predictions (
			node_id             VARCHAR NOT NULL,
			ts                  TIMESTAMP NOT NULL,
			horizon_hours       INTEGER NOT NULL,
			temperature_c       FLOAT,
			humidity_pct        FLOAT,
			rainfall_mm         FLOAT,
			rainfall_confidence FLOAT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_hourly_predictions_node_ts ON hourly_predictions (node_id, ts);`,

		// 4. Daily forecast predictions table
		`CREATE TABLE IF NOT EXISTS daily_predictions (
			node_id             VARCHAR NOT NULL,
			ts                  TIMESTAMP NOT NULL,
			horizon_days        INTEGER NOT NULL,
			min_temperature_c   FLOAT,
			max_temperature_c   FLOAT,
			min_humidity_pct    FLOAT,
			max_humidity_pct    FLOAT,
			rainfall_mm         FLOAT,
			rainfall_confidence FLOAT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_daily_predictions_node_ts ON daily_predictions (node_id, ts);`,
	}

	for _, q := range queries {
		if _, err := c.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("exec schema statement failed: %w (query: %s)", err, q)
		}
	}

	return nil
}
