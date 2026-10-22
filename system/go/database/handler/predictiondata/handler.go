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

// New returns a handler for the PredictionDatabaseService.
func New(params HandlerParams) *Handler {
	return &Handler{
		log: params.Logger,
		con: params.Controller,
	}
}

// Register exposes this handler as the PredictionDatabaseService.
func (h *Handler) Register(s grpc.ServiceRegistrar) {
	protodatabasev1.RegisterPredictionDatabaseServiceServer(s, h)
}

// StoreHourlyPredictions stores a batch of hourly forecast predictions.
func (h *Handler) StoreHourlyPredictions(ctx context.Context, req *protodatabasev1.StoreHourlyPredictionsRequest) (*protodatabasev1.StoreHourlyPredictionsResponse, error) {
	predictions := make([]prediction.HourlyPrediction, 0, len(req.Predictions))
	for _, item := range req.Predictions {
		predictions = append(predictions, pbHourlyToModel(item))
	}
	stored, err := h.con.StoreHourlyPredictions(ctx, predictions)
	if err != nil {
		return nil, err
	}
	return &protodatabasev1.StoreHourlyPredictionsResponse{Stored: uint64(stored)}, nil
}

// StoreDailyPredictions stores a batch of daily forecast predictions.
func (h *Handler) StoreDailyPredictions(ctx context.Context, req *protodatabasev1.StoreDailyPredictionsRequest) (*protodatabasev1.StoreDailyPredictionsResponse, error) {
	predictions := make([]prediction.DailyPrediction, 0, len(req.Predictions))
	for _, item := range req.Predictions {
		predictions = append(predictions, pbDailyToModel(item))
	}
	stored, err := h.con.StoreDailyPredictions(ctx, predictions)
	if err != nil {
		return nil, err
	}
	return &protodatabasev1.StoreDailyPredictionsResponse{Stored: uint64(stored)}, nil
}

// GetForecast returns the latest hourly and daily forecast predictions for a node.
func (h *Handler) GetForecast(ctx context.Context, req *protodatabasev1.GetForecastRequest) (*protodatabasev1.GetForecastResponse, error) {
	forecast, err := h.con.GetForecast(ctx, req.NodeId)
	if err != nil {
		return nil, err
	}
	response := &protodatabasev1.GetForecastResponse{
		HourlyPredictions: make([]*protodatabasev1.HourlyPrediction, 0, len(forecast.Hourly)),
		DailyPredictions:  make([]*protodatabasev1.DailyPrediction, 0, len(forecast.Daily)),
	}
	for _, item := range forecast.Hourly {
		response.HourlyPredictions = append(response.HourlyPredictions, modelHourlyToPB(item))
	}
	for _, item := range forecast.Daily {
		response.DailyPredictions = append(response.DailyPredictions, modelDailyToPB(item))
	}
	return response, nil
}
