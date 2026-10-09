package health

import (
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Handler struct {
	log *zap.Logger
}

type HandlerParams struct {
	fx.In

	Logger *zap.Logger
}

func New(params HandlerParams) (*Handler, error) {
	return &Handler{log: params.Logger}, nil
}

func (*Handler) Pattern() string {
	return "/health"
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Received request", zap.String("url", r.URL.String()))

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// GRPCHandler is the gRPC health service registrar.
type GRPCHandler struct{}

// NewGRPC returns the gRPC health service registrar.
func NewGRPC() *GRPCHandler {
	return &GRPCHandler{}
}

// Register exposes the gRPC health service.
func (*GRPCHandler) Register(s grpc.ServiceRegistrar) {
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
}
