package inferenceserver

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/fx"
	"go.uber.org/zap"

	grpcclient "weather-station/common/client/grpc"
	protoinferencev1 "weather-station/gen/go/weather/inference/v1"
)

var (
	// ErrNilRequest is returned when attempting to call inference with a nil request.
	ErrNilRequest = errors.New("request is nil")
	// ErrNoClient is returned when neither a client nor client factory is provided.
	ErrNoClient = errors.New("client or client factory is required")
)

// Gateway provides access to the inference server gRPC service.
type Gateway struct {
	log    *zap.Logger
	client protoinferencev1.InferenceServiceClient
}

// GatewayParams defines dependencies for Gateway.
type GatewayParams struct {
	fx.In

	Logger        *zap.Logger
	Config        Config
	ClientFactory *grpcclient.ClientFactory                  `optional:"true"`
	Client        protoinferencev1.InferenceServiceClient   `optional:"true"`
}

// New creates a new inference server Gateway instance.
func New(params GatewayParams) (*Gateway, error) {
	if params.Client != nil {
		return &Gateway{
			log:    params.Logger,
			client: params.Client,
		}, nil
	}

	if params.ClientFactory == nil {
		return nil, ErrNoClient
	}

	conn, err := params.ClientFactory.NewConnection(params.Config.Address)
	if err != nil {
		return nil, fmt.Errorf("connect to inference server %q: %w", params.Config.Address, err)
	}

	return &Gateway{
		log:    params.Logger,
		client: protoinferencev1.NewInferenceServiceClient(conn),
	}, nil
}

// PredictWeather sends time series observations and returns forecasted predictions.
func (g *Gateway) PredictWeather(ctx context.Context, req *protoinferencev1.PredictWeatherRequest) (*protoinferencev1.PredictWeatherResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	res, err := g.client.PredictWeather(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("predict weather for node %q: %w", req.GetNodeId(), err)
	}

	return res, nil
}

// GetModelInfo returns metadata and bounds from the active inference model.
func (g *Gateway) GetModelInfo(ctx context.Context) (*protoinferencev1.GetModelInfoResponse, error) {
	res, err := g.client.GetModelInfo(ctx, &protoinferencev1.GetModelInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("get model info: %w", err)
	}

	return res, nil
}
