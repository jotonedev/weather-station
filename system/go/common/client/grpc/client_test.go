package grpc

import (
	"context"
	"testing"
	"weather-station/common/internal/mocks"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestHealthClientMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create the mock client
	mockHealthClient := mocks.NewMockHealthClient(ctrl)

	// Define expectation
	expectedResponse := &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}

	mockHealthClient.EXPECT().
		Check(gomock.Any(), gomock.Any()).
		Return(expectedResponse, nil)

	// Use the mock
	resp, err := mockHealthClient.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("Expected status SERVING, got %v", resp.Status)
	}
}
