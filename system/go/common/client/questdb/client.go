package questdb

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Client struct {
	pool *pgxpool.Pool
	log  *zap.Logger
}

func NewClient(lc fx.Lifecycle, cfg *Config, log *zap.Logger) (*Client, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, err
	}

	c := &Client{pool: pool, log: log}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			timeout := cfg.ConnectTimeout
			pingCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			if err := pool.Ping(pingCtx); err != nil {
				// Fail startup fast so a bad QUESTDB_DSN surfaces immediately,
				// mirroring the strict health check in client/grpc.
				c.log.Error("QuestDB ping failed", zap.Error(err))
				return err
			}
			c.log.Info("QuestDB ping successful")
			return nil
		},
		OnStop: func(_ context.Context) error {
			c.log.Info("Closing QuestDB pool")
			pool.Close()
			return nil
		},
	})

	return c, nil
}

// Pool exposes the underlying pool for repository query layers.
func (c *Client) Pool() *pgxpool.Pool {
	return c.pool
}

// Ping verifies connectivity.
func (c *Client) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

// Exec runs a statement without returning rows.
func (c *Client) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return c.pool.Exec(ctx, sql, args...)
}

// Query runs a statement returning rows. Caller must close rows.
func (c *Client) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return c.pool.Query(ctx, sql, args...)
}

// CopyFrom performs bulk ingest (used for StoreReadings batches).
func (c *Client) CopyFrom(ctx context.Context, table pgx.Identifier, columns []string, source pgx.CopyFromSource) (int64, error) {
	return c.pool.CopyFrom(ctx, table, columns, source)
}
