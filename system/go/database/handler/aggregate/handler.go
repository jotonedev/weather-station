package aggregate

import (
	"context"

	"weather-station/common/pb"
	controller "weather-station/database/controller/aggregate"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Handler struct {
	log *zap.Logger
	con *controller.Controller
	protodatabasev1.UnimplementedReadingAggregateServiceServer
}

type HandlerParams struct {
	fx.In

	Logger     *zap.Logger
	Controller *controller.Controller
}

// New returns a handler for the ReadingAggregateService.
func New(params HandlerParams) *Handler {
	return &Handler{
		log: params.Logger,
		con: params.Controller,
	}
}

// Register exposes this handler as the ReadingAggregateService.
func (h *Handler) Register(s grpc.ServiceRegistrar) {
	protodatabasev1.RegisterReadingAggregateServiceServer(s, h)
}

// GetClimateAggregates returns climate aggregates per time bucket for a node.
func (h *Handler) GetClimateAggregates(ctx context.Context, req *protodatabasev1.GetClimateAggregatesRequest) (*protodatabasev1.GetClimateAggregatesResponse, error) {
	aggregates, err := h.con.GetClimateAggregates(ctx, req.NodeId, pb.FromTime(req.StartTime), pb.FromTime(req.EndTime), req.BucketSeconds)
	if err != nil {
		return nil, err
	}
	response := &protodatabasev1.GetClimateAggregatesResponse{Aggregates: make([]*protodatabasev1.ClimateAggregate, 0, len(aggregates))}
	for _, aggregate := range aggregates {
		response.Aggregates = append(response.Aggregates, modelClimateToPB(aggregate))
	}
	return response, nil
}

// GetAirQualityAggregates returns air quality aggregates per time bucket for a node.
func (h *Handler) GetAirQualityAggregates(ctx context.Context, req *protodatabasev1.GetAirQualityAggregatesRequest) (*protodatabasev1.GetAirQualityAggregatesResponse, error) {
	aggregates, err := h.con.GetAirQualityAggregates(ctx, req.NodeId, pb.FromTime(req.StartTime), pb.FromTime(req.EndTime), req.BucketSeconds)
	if err != nil {
		return nil, err
	}
	response := &protodatabasev1.GetAirQualityAggregatesResponse{Aggregates: make([]*protodatabasev1.AirQualityAggregate, 0, len(aggregates))}
	for _, aggregate := range aggregates {
		response.Aggregates = append(response.Aggregates, modelAirQualityToPB(aggregate))
	}
	return response, nil
}
