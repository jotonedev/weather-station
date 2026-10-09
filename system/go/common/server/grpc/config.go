package grpc

import "os"

// Config holds gRPC server settings.
type Config struct {
	// Address is the host:port the server listens on.
	Address string
}

const defaultAddress = "0.0.0.0:8090"

// NewConfig reads GRPC_ADDRESS from the environment, falling back to the default.
func NewConfig() Config {
	if address := os.Getenv("GRPC_ADDRESS"); address != "" {
		return Config{Address: address}
	}
	return Config{Address: defaultAddress}
}
