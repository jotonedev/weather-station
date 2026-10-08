package questdb

import (
	"os"
	"strconv"
	"time"
)

// Config holds connection settings for the QuestDB client.
type Config struct {
	// DSN is a postgres connection string, e.g.
	// postgres://admin:quest@localhost:8812/qdb
	DSN string
	// MaxConns caps the pgx pool.
	MaxConns int32
	// MinConns is the floor of idle connections to keep.
	MinConns int32
	// ConnectTimeout bounds the initial Ping during fx OnStart.
	ConnectTimeout time.Duration
}

const (
	defaultDSN            = "postgres://admin:quest@localhost:8812/qdb"
	defaultMaxConns int32 = 4
	defaultMinConns int32 = 1
	defaultTimeout        = 5 * time.Second
)

// NewConfig reads QUESTDB_DSN, QUESTDB_MAX_CONNS, QUESTDB_MIN_CONNS,
// QUESTDB_CONNECT_TIMEOUT_MS from the environment with sane defaults.
func NewConfig() (*Config, error) {
	cfg := &Config{
		DSN:            defaultDSN,
		MaxConns:       defaultMaxConns,
		MinConns:       defaultMinConns,
		ConnectTimeout: defaultTimeout,
	}

	if v := os.Getenv("QUESTDB_DSN"); v != "" {
		cfg.DSN = v
	}
	if v := os.Getenv("QUESTDB_MAX_CONNS"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, err
		}
		cfg.MaxConns = int32(n)
	}
	if v := os.Getenv("QUESTDB_MIN_CONNS"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, err
		}
		cfg.MinConns = int32(n)
	}
	if v := os.Getenv("QUESTDB_CONNECT_TIMEOUT_MS"); v != "" {
		ms, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		cfg.ConnectTimeout = time.Duration(ms) * time.Millisecond
	}

	return cfg, nil
}
