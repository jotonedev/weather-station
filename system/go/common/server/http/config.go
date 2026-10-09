package http

import "os"

// Config holds HTTP server settings.
type Config struct {
	// Address is the host:port the server listens on.
	Address string
}

const defaultAddress = "0.0.0.0:8080"

// NewConfig reads HTTP_ADDRESS from the environment, falling back to the default.
func NewConfig() Config {
	if address := os.Getenv("HTTP_ADDRESS"); address != "" {
		return Config{Address: address}
	}
	return Config{Address: defaultAddress}
}
