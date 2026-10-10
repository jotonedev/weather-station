package inferenceserver

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	protoinferencev1 "weather-station/gen/go/weather/inference/v1"
	"weather-station/inference-job/internal/mocks"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name        string
		envKey      string
		envVal      string
		wantAddress string
	}{
		{
			name:        "default address when no env set",
			wantAddress: "ws-inference-server:8090",
		},
		{
			name:        "inference server address with port",
			envKey:      "INFERENCE_SERVER_ADDRESS",
			envVal:      "remote-host:9090",
			wantAddress: "remote-host:9090",
		},
		{
			name:        "inference server address without port appends default port",
			envKey:      "INFERENCE_SERVER_ADDRESS",
			envVal:      "custom-host",
			wantAddress: "custom-host:8090",
		},
		{
			name:        "inference server grpc address env",
			envKey:      "INFERENCE_SERVER_GRPC_ADDRESS",
			envVal:      "grpc-host:50051",
			wantAddress: "grpc-host:50051",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}
			cfg := NewConfig()
			if cfg.Address != tt.wantAddress {
				t.Fatalf("Address = %q, want %q", cfg.Address, tt.wantAddress)
			}
		})
	}
}

func TestGateway(t *testing.T) {
	logger := zap.NewNop()
	errCall := errors.New("rpc failed")

	tests := []struct {
		name       string
		req        *protoinferencev1.PredictWeatherRequest
		setup      func(client *mocks.MockInferenceServiceClient)
		wantErr    bool
		wantModel  string
	}{
		{
			name:    "nil request returns error",
			req:     nil,
			setup:   func(client *mocks.MockInferenceServiceClient) {},
			wantErr: true,
		},
		{
			name: "predict weather rpc error",
			req: &protoinferencev1.PredictWeatherRequest{
				NodeId: "node-1",
				Mode:   protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_24_HOURS,
			},
			setup: func(client *mocks.MockInferenceServiceClient) {
				client.EXPECT().
					PredictWeather(gomock.Any(), gomock.Any()).
					Return(nil, errCall)
			},
			wantErr: true,
		},
		{
			name: "predict weather success",
			req: &protoinferencev1.PredictWeatherRequest{
				NodeId: "node-1",
				Mode:   protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_24_HOURS,
			},
			setup: func(client *mocks.MockInferenceServiceClient) {
				client.EXPECT().
					PredictWeather(gomock.Any(), gomock.Any()).
					Return(&protoinferencev1.PredictWeatherResponse{
						ModelVersion: "chronos-2-bolt-base",
					}, nil)
			},
			wantErr:   false,
			wantModel: "chronos-2-bolt-base",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			client := mocks.NewMockInferenceServiceClient(ctrl)
			tt.setup(client)

			gw, err := New(GatewayParams{
				Logger: logger,
				Client: client,
			})
			if err != nil {
				t.Fatalf("unexpected New error: %v", err)
			}

			res, err := gw.PredictWeather(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("PredictWeather() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && res.GetModelVersion() != tt.wantModel {
				t.Fatalf("PredictWeather() model = %q, want %q", res.GetModelVersion(), tt.wantModel)
			}
		})
	}
}

func TestGatewayGetModelInfo(t *testing.T) {
	logger := zap.NewNop()
	errCall := errors.New("info failed")

	tests := []struct {
		name      string
		setup     func(client *mocks.MockInferenceServiceClient)
		wantErr   bool
		wantName  string
	}{
		{
			name: "rpc error",
			setup: func(client *mocks.MockInferenceServiceClient) {
				client.EXPECT().
					GetModelInfo(gomock.Any(), gomock.Any()).
					Return(nil, errCall)
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func(client *mocks.MockInferenceServiceClient) {
				client.EXPECT().
					GetModelInfo(gomock.Any(), gomock.Any()).
					Return(&protoinferencev1.GetModelInfoResponse{
						ModelName: "chronos-2",
					}, nil)
			},
			wantErr:  false,
			wantName: "chronos-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			client := mocks.NewMockInferenceServiceClient(ctrl)
			tt.setup(client)

			gw, err := New(GatewayParams{
				Logger: logger,
				Client: client,
			})
			if err != nil {
				t.Fatalf("unexpected New error: %v", err)
			}

			res, err := gw.GetModelInfo(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetModelInfo() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && res.GetModelName() != tt.wantName {
				t.Fatalf("GetModelInfo() name = %q, want %q", res.GetModelName(), tt.wantName)
			}
		})
	}
}
