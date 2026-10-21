package homepage

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"weather-station/frontend/controller/weatherdata"
	"weather-station/frontend/gateway/database"
	"weather-station/frontend/internal/mocks"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestHomepageControllerRender(t *testing.T) {
	temp := float32(21.5)
	forecastTemp := float32(17.2)
	forecastRain := float32(2.4)
	pm25 := int32(9)

	testCases := []struct {
		name          string
		requestedNode string
		setup         func(readingClient *mocks.MockReadingDatabaseServiceClient, predictionClient *mocks.MockPredictionDatabaseServiceClient)
		check         func(t *testing.T, content string)
	}{
		{
			name:          "renders no-stations empty state when no node is selected",
			requestedNode: "",
			setup:         func(readingClient *mocks.MockReadingDatabaseServiceClient, predictionClient *mocks.MockPredictionDatabaseServiceClient) {},
			check: func(t *testing.T, content string) {
				if !strings.Contains(content, "No Weather Stations Available") {
					t.Fatal("Render() output missing no-stations empty state")
				}
				if strings.Contains(content, "Station: <strong>") {
					t.Fatal("Render() output should not show a station subtitle when no node is selected")
				}
			},
		},
		{
			name:          "renders current reading, air quality badge, and forecast cards",
			requestedNode: "station-test",
			setup: func(readingClient *mocks.MockReadingDatabaseServiceClient, predictionClient *mocks.MockPredictionDatabaseServiceClient) {
				readingClient.EXPECT().
					GetLatestReadings(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.GetLatestReadingsResponse{
						Readings: []*protodatabasev1.Reading{
							{
								NodeId:      "station-test",
								Temperature: &temp,
								Pm25:        &pm25,
							},
						},
					}, nil)

				predictionClient.EXPECT().
					GetForecast(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.GetForecastResponse{
						Predictions: []*protodatabasev1.Prediction{
							{
								NodeId:       "station-test",
								HorizonHours: 6,
								TemperatureC: &forecastTemp,
								RainfallMm:   &forecastRain,
							},
						},
					}, nil)
			},
			check: func(t *testing.T, content string) {
				for _, want := range []string{
					"<!DOCTYPE html>",
					"/static/css/weather-icons.min.css",
					"/static/css/styles.css",
					"station-test",
					"21.5 °C",
					"Good",
					"Weather Forecast",
					"17.2 °C",
					"2.4 mm",
					"wi-showers",
					"Erik Flowers",
				} {
					if !strings.Contains(content, want) {
						t.Fatalf("Render() output missing %q", want)
					}
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockReading := mocks.NewMockReadingDatabaseServiceClient(ctrl)
			mockPrediction := mocks.NewMockPredictionDatabaseServiceClient(ctrl)
			testCase.setup(mockReading, mockPrediction)

			dbGateway, err := database.New(database.GatewayParams{
				Logger:           zap.NewNop(),
				Config:           database.Config{Address: "127.0.0.1:8090"},
				ReadingClient:    mockReading,
				PredictionClient: mockPrediction,
			})
			if err != nil {
				t.Fatalf("database.New() unexpected error = %v", err)
			}

			dataCtrl, err := weatherdata.New(weatherdata.ControllerParams{
				Logger:   zap.NewNop(),
				Database: dbGateway,
			})
			if err != nil {
				t.Fatalf("weatherdata.New() unexpected error = %v", err)
			}

			templateCtrl, err := New(ControllerParams{
				Logger:      zap.NewNop(),
				WeatherData: dataCtrl,
			})
			if err != nil {
				t.Fatalf("homepage.New() unexpected error = %v", err)
			}

			rendered, err := templateCtrl.Render(context.Background(), testCase.requestedNode)
			if err != nil {
				t.Fatalf("Render() unexpected error = %v", err)
			}

			testCase.check(t, string(rendered))
		})
	}
}
