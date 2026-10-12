package client

import (
	"os"
	"strconv"
	"time"
)

// Config holds connection and tuning settings for the embedded DuckDB database.
type Config struct {
	// Path is the filesystem path to the DuckDB file, or "" / ":memory:" for an in-memory database.
	Path string
	// MaxMemory is the maximum memory DuckDB can use (e.g. "128MB", "256MB").
	MaxMemory string
	// Threads is the maximum number of worker threads DuckDB will spawn.
	Threads int
	// ConnectTimeout bounds the initial ping check during fx OnStart.
	ConnectTimeout time.Duration
}

const (
	defaultPath      = "/data/weather.duckdb"
	defaultMaxMemory = "128MB"
	defaultThreads   = 2
	defaultTimeout   = 5 * time.Second
)

// NewConfig reads DUCKDB_PATH, DUCKDB_MAX_MEMORY, DUCKDB_THREADS from the environment with sane defaults.
func NewConfig() (*Config, error) {
	cfg := &Config{
		Path:           defaultPath,
		MaxMemory:      defaultMaxMemory,
		Threads:        defaultThreads,
		ConnectTimeout: defaultTimeout,
	}

	if v := os.Getenv("DUCKDB_PATH"); v != "" {
		cfg.Path = v
	}
	if v := os.Getenv("DUCKDB_MAX_MEMORY"); v != "" {
		cfg.MaxMemory = v
	}
	if v := os.Getenv("DUCKDB_THREADS"); v != "" {
		threads, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		if threads > 0 {
			cfg.Threads = threads
		}
	}
	if v := os.Getenv("DUCKDB_CONNECT_TIMEOUT_MS"); v != "" {
		ms, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		cfg.ConnectTimeout = time.Duration(ms) * time.Millisecond
	}

	return cfg, nil
}
