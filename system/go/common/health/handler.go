package health

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// HealthHandler serves the HTTP liveness probe endpoint.
type HealthHandler struct {
	log *zap.Logger
}

// HealthHandlerParams defines dependencies for HealthHandler.
type HealthHandlerParams struct {
	fx.In

	Logger *zap.Logger
}

// Handler is an alias for HealthHandler to preserve backward compatibility.
type Handler = HealthHandler

// HandlerParams is an alias for HealthHandlerParams to preserve backward compatibility.
type HandlerParams = HealthHandlerParams

// New returns a new HealthHandler instance.
func New(params HandlerParams) (*Handler, error) {
	return NewHealth(params)
}

// NewHealth returns a new HealthHandler instance.
func NewHealth(params HealthHandlerParams) (*HealthHandler, error) {
	return &HealthHandler{log: params.Logger}, nil
}

// Pattern reports the path for the liveness endpoint.
func (*HealthHandler) Pattern() string {
	return "/health"
}

// ServeHTTP writes ok to signal service liveness.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Received health request", zap.String("url", r.URL.String()))

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// ReadyHandler serves the HTTP readiness probe endpoint.
type ReadyHandler struct {
	log      *zap.Logger
	checkers []Checker
}

// ReadyHandlerParams defines dependencies for ReadyHandler.
type ReadyHandlerParams struct {
	fx.In

	Logger   *zap.Logger
	Checkers []Checker `group:"readiness-checkers"`
}

// NewReady returns a new ReadyHandler instance.
func NewReady(params ReadyHandlerParams) (*ReadyHandler, error) {
	return &ReadyHandler{
		log:      params.Logger,
		checkers: params.Checkers,
	}, nil
}

// Pattern reports the path for the readiness endpoint.
func (*ReadyHandler) Pattern() string {
	return "/ready"
}

// ServeHTTP evaluates all registered readiness checkers.
func (h *ReadyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Received ready request", zap.String("url", r.URL.String()))

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	for _, checker := range h.checkers {
		if err := checker.Check(ctx); err != nil {
			h.log.Warn("Readiness check failed", zap.String("checker", checker.Name()), zap.Error(err))
			http.Error(w, fmt.Sprintf("%s: %v", checker.Name(), err), http.StatusServiceUnavailable)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// GRPCHandler registers gRPC health checks.
type GRPCHandler struct {
	base     *health.Server
	checkers []Checker
}

type grpcHealthServer struct {
	grpc_health_v1.HealthServer
	checkers []Checker
}

func (s *grpcHealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	if req != nil && req.Service == "ready" {
		for _, checker := range s.checkers {
			if err := checker.Check(ctx); err != nil {
				return &grpc_health_v1.HealthCheckResponse{
					Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
				}, nil
			}
		}
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_SERVING,
		}, nil
	}
	return s.HealthServer.Check(ctx, req)
}

// NewGRPC returns the gRPC health service registrar.
func NewGRPC(lc fx.Lifecycle, params ReadyHandlerParams) *GRPCHandler {
	base := health.NewServer()
	base.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			base.Shutdown()
			return nil
		},
	})

	return &GRPCHandler{
		base:     base,
		checkers: params.Checkers,
	}
}

// Register exposes the gRPC health service.
func (h *GRPCHandler) Register(s grpc.ServiceRegistrar) {
	grpc_health_v1.RegisterHealthServer(s, &grpcHealthServer{
		HealthServer: h.base,
		checkers:     h.checkers,
	})
}
