package prediction

import (
	"context"
	"testing"
	"time"

	duckdb "weather-station/database/client"

	"go.uber.org/zap"
)

func TestExtractHourlyValues(t *testing.T) {
	timestamp := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	rainfall := float32(1.5)
	tests := []struct {
		name       string
		prediction HourlyPrediction
		wantLen    int
		wantNode   string
	}{
		{
			name:       "present metrics",
			prediction: HourlyPrediction{NodeID: "n1", Ts: timestamp, HorizonHours: 6, RainfallMm: &rainfall},
			wantLen:    7,
			wantNode:   "n1",
		},
		{
			name:       "absent metrics are nil",
			prediction: HourlyPrediction{NodeID: "n2", Ts: timestamp, HorizonHours: 24},
			wantLen:    7,
			wantNode:   "n2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := extractHourlyValues(tt.prediction)
			if len(values) != tt.wantLen {
				t.Fatalf("len(values) = %d, want %d", len(values), tt.wantLen)
			}
			if values[0] != tt.wantNode {
				t.Errorf("node_id = %v, want %q", values[0], tt.wantNode)
			}
			if values[2] != tt.prediction.HorizonHours {
				t.Errorf("horizon = %v, want %d", values[2], tt.prediction.HorizonHours)
			}
		})
	}
}

func TestExtractDailyValues(t *testing.T) {
	timestamp := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	rainfall := float32(3.2)
	prediction := DailyPrediction{NodeID: "n1", Ts: timestamp, HorizonDays: 2, RainfallMm: &rainfall}
	values := extractDailyValues(prediction)

	if len(values) != 9 {
		t.Fatalf("len(values) = %d, want 9", len(values))
	}
	if values[0] != "n1" {
		t.Errorf("node_id = %v, want %q", values[0], "n1")
	}
	if values[2] != prediction.HorizonDays {
		t.Errorf("horizon = %v, want %d", values[2], prediction.HorizonDays)
	}
	if value, ok := values[7].(float32); !ok || value != rainfall {
		t.Errorf("rainfall = %v, want %v", values[7], rainfall)
	}
	for _, columnIndex := range []int{3, 4, 5, 6, 8} {
		if values[columnIndex] != nil {
			t.Errorf("index %d should be nil, got %v", columnIndex, values[columnIndex])
		}
	}
}

func TestStoreHourlyPredictionsEmpty(t *testing.T) {
	repo := &Repository{client: nil, log: zap.NewNop()}
	stored, err := repo.StoreHourlyPredictions(context.Background(), nil)
	if err != nil {
		t.Fatalf("StoreHourlyPredictions(empty): %v", err)
	}
	if stored != 0 {
		t.Errorf("stored = %d, want 0", stored)
	}
}

func TestStoreDailyPredictionsEmpty(t *testing.T) {
	repo := &Repository{client: nil, log: zap.NewNop()}
	stored, err := repo.StoreDailyPredictions(context.Background(), nil)
	if err != nil {
		t.Fatalf("StoreDailyPredictions(empty): %v", err)
	}
	if stored != 0 {
		t.Errorf("stored = %d, want 0", stored)
	}
}

func TestStoreAndGetHourlyPredictions(testingT *testing.T) {
	ctx := context.Background()
	dbClient, clientErr := duckdb.NewTestClient(ctx)
	if clientErr != nil {
		testingT.Fatalf("failed to create test database client: %v", clientErr)
	}
	defer dbClient.DB().Close()

	repo := New(dbClient, zap.NewNop())
	issueTime := time.Date(2026, 10, 22, 13, 0, 0, 0, time.UTC)
	temperatureFirst := float32(18.5)
	temperatureSecond := float32(19.0)

	initialBatch := []HourlyPrediction{
		{NodeID: "station-alpha", Ts: issueTime, HorizonHours: 1, TemperatureC: &temperatureFirst},
		{NodeID: "station-alpha", Ts: issueTime, HorizonHours: 2, TemperatureC: &temperatureFirst},
	}

	storedCount, storeErr := repo.StoreHourlyPredictions(ctx, initialBatch)
	if storeErr != nil {
		testingT.Fatalf("initial StoreHourlyPredictions failed: %v", storeErr)
	}
	if storedCount != 2 {
		testingT.Errorf("storedCount = %d, want 2", storedCount)
	}

	retryBatch := []HourlyPrediction{
		{NodeID: "station-alpha", Ts: issueTime, HorizonHours: 1, TemperatureC: &temperatureSecond},
		{NodeID: "station-alpha", Ts: issueTime, HorizonHours: 2, TemperatureC: &temperatureSecond},
	}

	retriedCount, retryErr := repo.StoreHourlyPredictions(ctx, retryBatch)
	if retryErr != nil {
		testingT.Fatalf("retry StoreHourlyPredictions failed: %v", retryErr)
	}
	if retriedCount != 2 {
		testingT.Errorf("retriedCount = %d, want 2", retriedCount)
	}

	forecast, queryErr := repo.GetForecast(ctx, "station-alpha")
	if queryErr != nil {
		testingT.Fatalf("GetForecast failed: %v", queryErr)
	}
	if len(forecast.Hourly) != 2 {
		testingT.Fatalf("len(forecast.Hourly) = %d, want 2", len(forecast.Hourly))
	}
	if *forecast.Hourly[0].TemperatureC != temperatureSecond {
		testingT.Errorf("forecast.Hourly[0].TemperatureC = %v, want %v", *forecast.Hourly[0].TemperatureC, temperatureSecond)
	}
}

func TestStoreAndGetDailyPredictions(testingT *testing.T) {
	ctx := context.Background()
	dbClient, clientErr := duckdb.NewTestClient(ctx)
	if clientErr != nil {
		testingT.Fatalf("failed to create test database client: %v", clientErr)
	}
	defer dbClient.DB().Close()

	repo := New(dbClient, zap.NewNop())
	issueTime := time.Date(2026, 10, 22, 13, 0, 0, 0, time.UTC)
	temperatureMin := float32(12.0)
	temperatureMax := float32(22.5)

	dailyBatch := []DailyPrediction{
		{NodeID: "station-alpha", Ts: issueTime, HorizonDays: 1, MinTemperatureC: &temperatureMin, MaxTemperatureC: &temperatureMax},
		{NodeID: "station-alpha", Ts: issueTime, HorizonDays: 2, MinTemperatureC: &temperatureMin, MaxTemperatureC: &temperatureMax},
	}

	storedCount, storeErr := repo.StoreDailyPredictions(ctx, dailyBatch)
	if storeErr != nil {
		testingT.Fatalf("StoreDailyPredictions failed: %v", storeErr)
	}
	if storedCount != 2 {
		testingT.Errorf("storedCount = %d, want 2", storedCount)
	}

	forecast, queryErr := repo.GetForecast(ctx, "station-alpha")
	if queryErr != nil {
		testingT.Fatalf("GetForecast failed: %v", queryErr)
	}
	if len(forecast.Daily) != 2 {
		testingT.Fatalf("len(forecast.Daily) = %d, want 2", len(forecast.Daily))
	}
	if *forecast.Daily[0].MaxTemperatureC != temperatureMax {
		testingT.Errorf("forecast.Daily[0].MaxTemperatureC = %v, want %v", *forecast.Daily[0].MaxTemperatureC, temperatureMax)
	}
}
