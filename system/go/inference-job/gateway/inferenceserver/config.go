package inferenceserver

import (
	"os"
	"strings"
)

// Config holds inference server gateway settings.
type Config struct {
	// Address is the host and port of the inference gRPC service.
	Address string
}

const (
	defaultHost    = "ws-inference-server"
	defaultPort    = "8090"
	defaultAddress = defaultHost + ":" + defaultPort
)

// NewConfig reads INFERENCE_SERVER_ADDRESS or INFERENCE_SERVER_GRPC_ADDRESS from the environment.
// It falls back to the default service address ws-inference-server:8090 if neither is set.
func NewConfig() Config {
	address := defaultAddress
	if env := os.Getenv("INFERENCE_SERVER_ADDRESS"); env != "" {
		address = env
	} else if env := os.Getenv("INFERENCE_SERVER_GRPC_ADDRESS"); env != "" {
		address = env
	}

	if !strings.Contains(address, ":") {
		address = address + ":" + defaultPort
	}

	return Config{Address: address}
}
