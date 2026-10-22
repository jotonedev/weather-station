package weatherdata

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"weather-station/common/pb"
	"weather-station/frontend/gateway/database"
	"weather-station/frontend/internal/mocks"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestGetDashboardData(t *testing.T) {
	temp := float32(23.4)
	forecastTemp := float32(19.5)
	dailyForecastTemp := float32(16.0)
	issueTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name          string
		requestedNode string
		setup         func(readingClient *mocks.MockReadingDatabaseServiceClient, predictionClient *mocks.MockPredictionDatabaseServiceClient, nodeClient *mocks.MockNodeDatabaseServiceClient)
		check         func(t *testing.T, data DashboardData)
	}{
		{
			name:          "displays latest readings and forecast when available",
			requestedNode: "station-beta",
			setup: func(readingClient *mocks.MockReadingDatabaseServiceClient, predictionClient *mocks.MockPredictionDatabaseServiceClient, nodeClient *mocks.MockNodeDatabaseServiceClient) {
				nodeClient.EXPECT().
					ListNodeCoordinates(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.ListNodeCoordinatesResponse{
						Nodes: []*protodatabasev1.NodeCoordinates{
							{NodeId: "station-alpha"},
							{NodeId: "station-beta"},
						},
					}, nil)

				readingClient.EXPECT().
					GetLatestReadings(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, in *protodatabasev1.GetLatestReadingsRequest, opts ...any) (*protodatabasev1.GetLatestReadingsResponse, error) {
						if in.NodeId != "station-beta" {
							t.Fatalf("NodeId = %q, want station-beta", in.NodeId)
						}
						return &protodatabasev1.GetLatestReadingsResponse{
							Readings: []*protodatabasev1.Reading{
								{
									NodeId:      "station-beta",
									Temperature: &temp,
								},
							},
						}, nil
					})

				predictionClient.EXPECT().
					GetForecast(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.GetForecastResponse{
						HourlyPredictions: []*protodatabasev1.HourlyPrediction{
							{
								NodeId:       "station-beta",
								IssueTime:    pb.Time(issueTime),
								HorizonHours: 1,
								TemperatureC: &forecastTemp,
							},
						},
						DailyPredictions: []*protodatabasev1.DailyPrediction{
							{
								NodeId:          "station-beta",
								IssueTime:       pb.Time(issueTime),
								HorizonDays:     1,
								MaxTemperatureC: &dailyForecastTemp,
							},
						},
					}, nil)
			},
			check: func(t *testing.T, data DashboardData) {
				if data.SelectedNodeID != "station-beta" {
					t.Fatalf("SelectedNodeID = %q, want station-beta", data.SelectedNodeID)
				}
				if len(data.AvailableNodes) != 2 {
					t.Fatalf("AvailableNodes length = %d, want 2", len(data.AvailableNodes))
				}
				if data.LatestReading == nil {
					t.Fatal("expected LatestReading to not be nil")
				}
				if data.LatestReading.Temperature != "23.4 °C" {
					t.Fatalf("Temperature = %q, want 23.4 °C", data.LatestReading.Temperature)
				}
				if len(data.Forecast) != 2 {
					t.Fatalf("Forecast length = %d, want 2", len(data.Forecast))
				}
				if len(data.HourlyForecast) != 1 || data.HourlyForecast[0].Temperature != "19.5 °C" {
					t.Fatalf("HourlyForecast = %+v, want 1 item with 19.5 °C", data.HourlyForecast)
				}
				if len(data.DailyForecast) != 1 || data.DailyForecast[0].MaxTemperature != "16.0 °C" {
					t.Fatalf("DailyForecast = %+v, want 1 item with 16.0 °C max", data.DailyForecast)
				}
				if data.HourlyIssuedAtISO != "2026-10-10T12:00:00Z" {
					t.Fatalf("HourlyIssuedAtISO = %q, want 2026-10-10T12:00:00Z", data.HourlyIssuedAtISO)
				}
				if data.DailyIssuedAtISO != "2026-10-10T12:00:00Z" {
					t.Fatalf("DailyIssuedAtISO = %q, want 2026-10-10T12:00:00Z", data.DailyIssuedAtISO)
				}
			},
		},
		{
			name:          "handles empty database gracefully with no stations",
			requestedNode: "",
			setup: func(readingClient *mocks.MockReadingDatabaseServiceClient, predictionClient *mocks.MockPredictionDatabaseServiceClient, nodeClient *mocks.MockNodeDatabaseServiceClient) {
				nodeClient.EXPECT().
					ListNodeCoordinates(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.ListNodeCoordinatesResponse{}, nil)
			},
			check: func(t *testing.T, data DashboardData) {
				if data.SelectedNodeID != "" {
					t.Fatalf("SelectedNodeID = %q, want empty", data.SelectedNodeID)
				}
				if !data.NoNodes {
					t.Fatal("expected NoNodes to be true")
				}
				if data.LatestReading != nil {
					t.Fatalf("LatestReading = %v, want nil", data.LatestReading)
				}
			},
		},
		{
			name:          "sets error message when database fails",
			requestedNode: "station-1",
			setup: func(readingClient *mocks.MockReadingDatabaseServiceClient, predictionClient *mocks.MockPredictionDatabaseServiceClient, nodeClient *mocks.MockNodeDatabaseServiceClient) {
				nodeClient.EXPECT().
					ListNodeCoordinates(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.ListNodeCoordinatesResponse{}, nil)

				readingClient.EXPECT().
					GetLatestReadings(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("connection failed"))
			},
			check: func(t *testing.T, data DashboardData) {
				if data.ErrorMessage == "" {
					t.Fatal("expected ErrorMessage to be populated")
				}
			},
		},
		{
			name:          "cuts off hourly and daily forecasts when confidence is below 10 percent",
			requestedNode: "station-beta",
			setup: func(readingClient *mocks.MockReadingDatabaseServiceClient, predictionClient *mocks.MockPredictionDatabaseServiceClient, nodeClient *mocks.MockNodeDatabaseServiceClient) {
				nodeClient.EXPECT().
					ListNodeCoordinates(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.ListNodeCoordinatesResponse{
						Nodes: []*protodatabasev1.NodeCoordinates{
							{NodeId: "station-beta"},
						},
					}, nil)

				temp := float32(20.0)
				readingClient.EXPECT().
					GetLatestReadings(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.GetLatestReadingsResponse{
						Readings: []*protodatabasev1.Reading{
							{
								NodeId:      "station-beta",
								Temperature: &temp,
							},
						},
					}, nil)

				c50 := float32(0.50)
				c10 := float32(0.10)
				c08 := float32(0.08)
				c80 := float32(0.80)
				c40 := float32(0.40)
				c05 := float32(0.05)
				c90 := float32(0.90)

				predictionClient.EXPECT().
					GetForecast(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.GetForecastResponse{
						HourlyPredictions: []*protodatabasev1.HourlyPrediction{
							{
								NodeId:             "station-beta",
								IssueTime:          pb.Time(issueTime),
								HorizonHours:       1,
								RainfallConfidence: &c50,
							},
							{
								NodeId:             "station-beta",
								IssueTime:          pb.Time(issueTime),
								HorizonHours:       2,
								RainfallConfidence: &c10,
							},
							{
								NodeId:             "station-beta",
								IssueTime:          pb.Time(issueTime),
								HorizonHours:       3,
								RainfallConfidence: &c08,
							},
							{
								NodeId:             "station-beta",
								IssueTime:          pb.Time(issueTime),
								HorizonHours:       4,
								RainfallConfidence: &c80,
							},
						},
						DailyPredictions: []*protodatabasev1.DailyPrediction{
							{
								NodeId:             "station-beta",
								IssueTime:          pb.Time(issueTime),
								HorizonDays:        1,
								RainfallConfidence: &c40,
							},
							{
								NodeId:             "station-beta",
								IssueTime:          pb.Time(issueTime),
								HorizonDays:        2,
								RainfallConfidence: &c05,
							},
							{
								NodeId:             "station-beta",
								IssueTime:          pb.Time(issueTime),
								HorizonDays:        3,
								RainfallConfidence: &c90,
							},
						},
					}, nil)
			},
			check: func(t *testing.T, data DashboardData) {
				if len(data.HourlyForecast) != 2 {
					t.Fatalf("HourlyForecast length = %d, want 2", len(data.HourlyForecast))
				}
				if data.HourlyForecast[0].HorizonHours != 1 || data.HourlyForecast[1].HorizonHours != 2 {
					t.Fatalf("HourlyForecast horizons = [%d, %d], want [1, 2]",
						data.HourlyForecast[0].HorizonHours, data.HourlyForecast[1].HorizonHours)
				}
				if len(data.DailyForecast) != 1 {
					t.Fatalf("DailyForecast length = %d, want 1", len(data.DailyForecast))
				}
				if data.DailyForecast[0].HorizonHours != 24 {
					t.Fatalf("DailyForecast horizon = %d, want 24", data.DailyForecast[0].HorizonHours)
				}
				if len(data.Forecast) != 3 {
					t.Fatalf("Forecast length = %d, want 3", len(data.Forecast))
				}
			},
		},
		{
			name:          "entire forecast below 10 percent confidence produces empty lists",
			requestedNode: "station-beta",
			setup: func(readingClient *mocks.MockReadingDatabaseServiceClient, predictionClient *mocks.MockPredictionDatabaseServiceClient, nodeClient *mocks.MockNodeDatabaseServiceClient) {
				nodeClient.EXPECT().
					ListNodeCoordinates(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.ListNodeCoordinatesResponse{
						Nodes: []*protodatabasev1.NodeCoordinates{
							{NodeId: "station-beta"},
						},
					}, nil)

				temp := float32(20.0)
				readingClient.EXPECT().
					GetLatestReadings(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.GetLatestReadingsResponse{
						Readings: []*protodatabasev1.Reading{
							{
								NodeId:      "station-beta",
								Temperature: &temp,
							},
						},
					}, nil)

				c05 := float32(0.05)
				c08 := float32(0.08)

				predictionClient.EXPECT().
					GetForecast(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.GetForecastResponse{
						HourlyPredictions: []*protodatabasev1.HourlyPrediction{
							{
								NodeId:             "station-beta",
								IssueTime:          pb.Time(issueTime),
								HorizonHours:       1,
								RainfallConfidence: &c05,
							},
						},
						DailyPredictions: []*protodatabasev1.DailyPrediction{
							{
								NodeId:             "station-beta",
								IssueTime:          pb.Time(issueTime),
								HorizonDays:        1,
								RainfallConfidence: &c08,
							},
						},
					}, nil)
			},
			check: func(t *testing.T, data DashboardData) {
				if len(data.HourlyForecast) != 0 {
					t.Fatalf("HourlyForecast length = %d, want 0", len(data.HourlyForecast))
				}
				if len(data.DailyForecast) != 0 {
					t.Fatalf("DailyForecast length = %d, want 0", len(data.DailyForecast))
				}
				if len(data.Forecast) != 0 {
					t.Fatalf("Forecast length = %d, want 0", len(data.Forecast))
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			readingClient := mocks.NewMockReadingDatabaseServiceClient(ctrl)
			predictionClient := mocks.NewMockPredictionDatabaseServiceClient(ctrl)
			nodeClient := mocks.NewMockNodeDatabaseServiceClient(ctrl)
			testCase.setup(readingClient, predictionClient, nodeClient)

			dbGateway, err := database.New(database.GatewayParams{
				Logger:           zap.NewNop(),
				Config:           database.Config{Address: "127.0.0.1:8090"},
				ReadingClient:    readingClient,
				PredictionClient: predictionClient,
				NodeClient:       nodeClient,
			})
			if err != nil {
				t.Fatalf("database.New() unexpected error = %v", err)
			}

			controller, err := New(ControllerParams{
				Logger:   zap.NewNop(),
				Database: dbGateway,
			})
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}

			data := controller.GetDashboardData(context.Background(), testCase.requestedNode)
			testCase.check(t, data)
		})
	}
}
