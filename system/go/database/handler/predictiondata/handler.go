package predictiondata

import (
	"context"

	"weather-station/database/repository/prediction"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	controller "weather-station/database/controller/predictiondata"
)

type Handler struct {
	log *zap.Logger
	con *controller.Controller
	protodatabasev1.UnimplementedPredictionDatabaseServiceServer
}

type HandlerParams struct {
	fx.In

	Logger     *zap.Logger
	Controller *controller.Controller
}

// New returns a registrar that exposes this handler as the PredictionDatabaseService.
func New(params HandlerParams) func(*grpc.Server) {
	handler := &Handler{
		log: params.Logger,
		con: params.Controller,
	}
	return func(s *grpc.Server) {
		protodatabasev1.RegisterPredictionDatabaseServiceServer(s, handler)
	}
}

// StorePredictions stores a batch of forecast predictions.
func (h *Handler) StorePredictions(ctx context.Context, req *protodatabasev1.StorePredictionsRequest) (*protodatabasev1.StorePredictionsResponse, error) {
	predictions := make([]prediction.Prediction, 0, len(req.Predictions))
	for _, item := range req.Predictions {
		predictions = append(predictions, pbPredictionToModel(item))
	}
	stored, err := h.con.StorePredictions(ctx, predictions)
	if err != nil {
		return nil, err
	}
	return &protodatabasev1.StorePredictionsResponse{Stored: uint64(stored)}, nil
}

// GetForecast returns the latest forecast predictions for a node.
func (h *Handler) GetForecast(ctx context.Context, req *protodatabasev1.GetForecastRequest) (*protodatabasev1.GetForecastResponse, error) {
	predictions, err := h.con.GetForecast(ctx, req.NodeId)
	if err != nil {
		return nil, err
	}
	response := &protodatabasev1.GetForecastResponse{Predictions: make([]*protodatabasev1.Prediction, 0, len(predictions))}
	for _, item := range predictions {
		response.Predictions = append(response.Predictions, modelPredictionToPB(item))
	}
	return response, nil
}
