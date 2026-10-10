package database

import (
	"os"
	"strings"
)

// Config holds database gateway settings.
type Config struct {
	// Address is the host and port of the database gRPC service.
	Address string
}

const (
	defaultHost    = "ws-database"
	defaultPort    = "8090"
	defaultAddress = defaultHost + ":" + defaultPort
)

// NewConfig reads DATABASE_ADDRESS or DATABASE_GRPC_ADDRESS from the environment.
// It falls back to the default service address ws-database:8090 if neither is set.
func NewConfig() Config {
	address := defaultAddress
	if env := os.Getenv("DATABASE_ADDRESS"); env != "" {
		address = env
	} else if env := os.Getenv("DATABASE_GRPC_ADDRESS"); env != "" {
		address = env
	}

	if !strings.Contains(address, ":") {
		address = address + ":" + defaultPort
	}

	return Config{Address: address}
}
