package readingdata

import (
	"context"

	"weather-station/common/pb"
	"weather-station/database/repository/timestorage"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	controller "weather-station/database/controller/readingdata"
)

type Handler struct {
	log *zap.Logger
	con *controller.Controller
	protodatabasev1.UnimplementedReadingDatabaseServiceServer
}

type HandlerParams struct {
	fx.In

	Logger     *zap.Logger
	Controller *controller.Controller
}

// New returns a registrar that exposes this handler as the ReadingDatabaseService.
func New(params HandlerParams) func(*grpc.Server) {
	handler := &Handler{
		log: params.Logger,
		con: params.Controller,
	}
	return func(s *grpc.Server) {
		protodatabasev1.RegisterReadingDatabaseServiceServer(s, handler)
	}
}

// StoreReadings stores a batch of raw readings.
func (h *Handler) StoreReadings(ctx context.Context, req *protodatabasev1.StoreReadingsRequest) (*protodatabasev1.StoreReadingsResponse, error) {
	readings := make([]timestorage.Reading, 0, len(req.Readings))
	for _, reading := range req.Readings {
		readings = append(readings, pbReadingToModel(reading))
	}
	stored, err := h.con.StoreReadings(ctx, readings)
	if err != nil {
		return nil, err
	}
	return &protodatabasev1.StoreReadingsResponse{Stored: uint64(stored)}, nil
}

// GetReadings streams raw readings for a node over a time range.
func (h *Handler) GetReadings(req *protodatabasev1.GetReadingsRequest, stream grpc.ServerStreamingServer[protodatabasev1.GetReadingsResponse]) error {
	readings, err := h.con.GetReadings(stream.Context(), req.NodeId, pb.FromTime(req.StartTime), pb.FromTime(req.EndTime))
	if err != nil {
		return err
	}
	for _, reading := range readings {
		if err := stream.Send(&protodatabasev1.GetReadingsResponse{Reading: modelReadingToPB(reading)}); err != nil {
			return err
		}
	}
	return nil
}

// GetLatestReadings returns the most recent raw readings for a node.
func (h *Handler) GetLatestReadings(ctx context.Context, req *protodatabasev1.GetLatestReadingsRequest) (*protodatabasev1.GetLatestReadingsResponse, error) {
	readings, err := h.con.GetLatestReadings(ctx, req.NodeId, int(req.Limit))
	if err != nil {
		return nil, err
	}
	response := &protodatabasev1.GetLatestReadingsResponse{Readings: make([]*protodatabasev1.Reading, 0, len(readings))}
	for _, reading := range readings {
		response.Readings = append(response.Readings, modelReadingToPB(reading))
	}
	return response, nil
}
