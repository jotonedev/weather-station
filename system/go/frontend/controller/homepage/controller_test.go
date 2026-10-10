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
	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockReadingDatabaseServiceClient(ctrl)

	temp := float32(21.5)
	mockClient.EXPECT().
		GetLatestReadings(gomock.Any(), gomock.Any()).
		Return(&protodatabasev1.GetLatestReadingsResponse{
			Readings: []*protodatabasev1.Reading{
				{NodeId: "station-test", Temperature: &temp},
			},
		}, nil)

	dbGateway, err := database.New(database.GatewayParams{
		Logger:        zap.NewNop(),
		Config:        database.Config{Address: "127.0.0.1:8090"},
		ReadingClient: mockClient,
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

	rendered, err := templateCtrl.Render(context.Background(), "station-test")
	if err != nil {
		t.Fatalf("Render() unexpected error = %v", err)
	}

	content := string(rendered)
	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Fatal("Render() output missing <!DOCTYPE html>")
	}
	if !strings.Contains(content, "station-test") {
		t.Fatal("Render() output missing station-test")
	}
	if !strings.Contains(content, "21.5 °C") {
		t.Fatal("Render() output missing 21.5 °C")
	}
}
