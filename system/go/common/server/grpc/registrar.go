package grpc

import "google.golang.org/grpc"

// Registrar registers a gRPC service with a server.
type Registrar interface {
	Register(grpc.ServiceRegistrar)
}
