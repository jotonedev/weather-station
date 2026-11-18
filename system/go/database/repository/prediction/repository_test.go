package prediction

import (
	"context"
	"fmt"
	"sync"
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
			wantLen:    15,
			wantNode:   "n1",
		},
		{
			name:       "absent metrics are nil",
			prediction: HourlyPrediction{NodeID: "n2", Ts: timestamp, HorizonHours: 24},
			wantLen:    15,
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

	if len(values) != 15 {
		t.Fatalf("len(values) = %d, want 15", len(values))
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
	for _, columnIndex := range []int{3, 4, 5, 6, 8, 9, 10, 11, 12, 13, 14} {
		if values[columnIndex] != nil {
			t.Errorf("index %d should be nil, got %v", columnIndex, values[columnIndex])
		}
	}
}

func TestStoreHourlyPredictionsEmpty(t *testing.T) {
	repo := &Repository{log: zap.NewNop()}
	stored, err := repo.StoreHourlyPredictions(context.Background(), nil)
	if err != nil {
		t.Fatalf("StoreHourlyPredictions(empty): %v", err)
	}
	if stored != 0 {
		t.Errorf("stored = %d, want 0", stored)
	}
}

func TestStoreDailyPredictionsEmpty(t *testing.T) {
	repo := &Repository{log: zap.NewNop()}
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
	tempP10 := float32(17.0)
	tempP90 := float32(21.0)
	rainProb := float32(0.35)
	pressure := float32(1013.2)

	initialBatch := []HourlyPrediction{
		{
			NodeID: "station-alpha", Ts: issueTime, HorizonHours: 1,
			TemperatureC: &temperatureFirst, TemperatureCP10: &tempP10, TemperatureCP90: &tempP90,
			RainProbability: &rainProb, PressureHpa: &pressure,
		},
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
		{
			NodeID: "station-alpha", Ts: issueTime, HorizonHours: 1,
			TemperatureC: &temperatureSecond, TemperatureCP10: &tempP10, TemperatureCP90: &tempP90,
			RainProbability: &rainProb, PressureHpa: &pressure,
		},
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
	if *forecast.Hourly[0].RainProbability != rainProb {
		testingT.Errorf("forecast.Hourly[0].RainProbability = %v, want %v", *forecast.Hourly[0].RainProbability, rainProb)
	}
	if *forecast.Hourly[0].PressureHpa != pressure {
		testingT.Errorf("forecast.Hourly[0].PressureHpa = %v, want %v", *forecast.Hourly[0].PressureHpa, pressure)
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

func TestInMemoryPredictions_OrderingAndMultiNode(t *testing.T) {
	ctx := context.Background()
	repo := New(nil, zap.NewNop())

	ts := time.Date(2026, 10, 22, 14, 0, 0, 0, time.UTC)
	tAlpha := float32(20.0)
	tBeta := float32(15.0)

	stored, err := repo.StoreHourlyPredictions(ctx, []HourlyPrediction{
		{NodeID: "station-alpha", Ts: ts, HorizonHours: 24, TemperatureC: &tAlpha},
		{NodeID: "station-alpha", Ts: ts, HorizonHours: 1, TemperatureC: &tAlpha},
		{NodeID: "station-alpha", Ts: ts, HorizonHours: 6, TemperatureC: &tAlpha},
		{NodeID: "station-beta", Ts: ts, HorizonHours: 12, TemperatureC: &tBeta},
		{NodeID: "station-beta", Ts: ts, HorizonHours: 3, TemperatureC: &tBeta},
	})
	if err != nil {
		t.Fatalf("StoreHourlyPredictions: %v", err)
	}
	if stored != 5 {
		t.Fatalf("stored = %d, want 5", stored)
	}

	storedDaily, err := repo.StoreDailyPredictions(ctx, []DailyPrediction{
		{NodeID: "station-alpha", Ts: ts, HorizonDays: 5, MaxTemperatureC: &tAlpha},
		{NodeID: "station-alpha", Ts: ts, HorizonDays: 1, MaxTemperatureC: &tAlpha},
		{NodeID: "station-alpha", Ts: ts, HorizonDays: 3, MaxTemperatureC: &tAlpha},
	})
	if err != nil {
		t.Fatalf("StoreDailyPredictions: %v", err)
	}
	if storedDaily != 3 {
		t.Fatalf("storedDaily = %d, want 3", storedDaily)
	}

	alphaForecast, err := repo.GetForecast(ctx, "station-alpha")
	if err != nil {
		t.Fatalf("GetForecast(station-alpha): %v", err)
	}
	if len(alphaForecast.Hourly) != 3 {
		t.Fatalf("len(alphaForecast.Hourly) = %d, want 3", len(alphaForecast.Hourly))
	}
	wantHourlyHorizons := []int32{1, 6, 24}
	for i, want := range wantHourlyHorizons {
		if alphaForecast.Hourly[i].HorizonHours != want {
			t.Errorf("alphaForecast.Hourly[%d].HorizonHours = %d, want %d", i, alphaForecast.Hourly[i].HorizonHours, want)
		}
	}
	wantDailyHorizons := []int32{1, 3, 5}
	if len(alphaForecast.Daily) != 3 {
		t.Fatalf("len(alphaForecast.Daily) = %d, want 3", len(alphaForecast.Daily))
	}
	for i, want := range wantDailyHorizons {
		if alphaForecast.Daily[i].HorizonDays != want {
			t.Errorf("alphaForecast.Daily[%d].HorizonDays = %d, want %d", i, alphaForecast.Daily[i].HorizonDays, want)
		}
	}

	betaForecast, err := repo.GetForecast(ctx, "station-beta")
	if err != nil {
		t.Fatalf("GetForecast(station-beta): %v", err)
	}
	if len(betaForecast.Hourly) != 2 || betaForecast.Hourly[0].HorizonHours != 3 || betaForecast.Hourly[1].HorizonHours != 12 {
		t.Fatalf("unexpected beta hourly forecast: %+v", betaForecast.Hourly)
	}
	if len(betaForecast.Daily) != 0 {
		t.Errorf("expected empty beta daily forecast, got %d", len(betaForecast.Daily))
	}

	missing, err := repo.GetForecast(ctx, "unknown-node")
	if err != nil {
		t.Fatalf("GetForecast(unknown-node): %v", err)
	}
	if len(missing.Hourly) != 0 || len(missing.Daily) != 0 {
		t.Errorf("expected empty forecast for unknown node, got %+v", missing)
	}
}

func TestInMemoryPredictions_LatestBatchRetention(t *testing.T) {
	ctx := context.Background()
	repo := New(nil, zap.NewNop())

	t1 := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 10, 22, 11, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

	v1 := float32(11.0)
	v2 := float32(12.0)
	v3 := float32(13.0)
	v3Retry := float32(13.5)

	if _, err := repo.StoreHourlyPredictions(ctx, []HourlyPrediction{
		{NodeID: "n1", Ts: t2, HorizonHours: 1, TemperatureC: &v2},
		{NodeID: "n1", Ts: t2, HorizonHours: 2, TemperatureC: &v2},
	}); err != nil {
		t.Fatalf("store t2 hourly: %v", err)
	}

	// Newer batch t3 replaces t2.
	if _, err := repo.StoreHourlyPredictions(ctx, []HourlyPrediction{
		{NodeID: "n1", Ts: t3, HorizonHours: 1, TemperatureC: &v3},
	}); err != nil {
		t.Fatalf("store t3 hourly: %v", err)
	}

	// Stale batch t1 must not overwrite t3.
	if _, err := repo.StoreHourlyPredictions(ctx, []HourlyPrediction{
		{NodeID: "n1", Ts: t1, HorizonHours: 1, TemperatureC: &v1},
	}); err != nil {
		t.Fatalf("store stale t1 hourly: %v", err)
	}

	forecast, err := repo.GetForecast(ctx, "n1")
	if err != nil {
		t.Fatalf("GetForecast: %v", err)
	}
	if len(forecast.Hourly) != 1 || !forecast.Hourly[0].Ts.Equal(t3) || *forecast.Hourly[0].TemperatureC != v3 {
		t.Fatalf("expected t3 batch with temp %v, got %+v", v3, forecast.Hourly)
	}

	// Same timestamp retry replaces t3.
	if _, err := repo.StoreHourlyPredictions(ctx, []HourlyPrediction{
		{NodeID: "n1", Ts: t3, HorizonHours: 1, TemperatureC: &v3Retry},
	}); err != nil {
		t.Fatalf("store retry t3 hourly: %v", err)
	}

	forecast, err = repo.GetForecast(ctx, "n1")
	if err != nil {
		t.Fatalf("GetForecast after retry: %v", err)
	}
	if len(forecast.Hourly) != 1 || *forecast.Hourly[0].TemperatureC != v3Retry {
		t.Fatalf("expected retry t3 batch with temp %v, got %+v", v3Retry, forecast.Hourly)
	}

	// Daily stale vs newer batch check.
	if _, err := repo.StoreDailyPredictions(ctx, []DailyPrediction{
		{NodeID: "n1", Ts: t3, HorizonDays: 1, MaxTemperatureC: &v3},
	}); err != nil {
		t.Fatalf("store t3 daily: %v", err)
	}
	if _, err := repo.StoreDailyPredictions(ctx, []DailyPrediction{
		{NodeID: "n1", Ts: t1, HorizonDays: 1, MaxTemperatureC: &v1},
	}); err != nil {
		t.Fatalf("store stale t1 daily: %v", err)
	}
	forecast, err = repo.GetForecast(ctx, "n1")
	if err != nil {
		t.Fatalf("GetForecast daily: %v", err)
	}
	if len(forecast.Daily) != 1 || !forecast.Daily[0].Ts.Equal(t3) || *forecast.Daily[0].MaxTemperatureC != v3 {
		t.Fatalf("expected t3 daily batch, got %+v", forecast.Daily)
	}
}

func TestInMemoryPredictions_SliceIsolation(t *testing.T) {
	ctx := context.Background()
	repo := New(nil, zap.NewNop())
	ts := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	temp := float32(20.0)

	input := []HourlyPrediction{{NodeID: "n1", Ts: ts, HorizonHours: 1, TemperatureC: &temp}}
	if _, err := repo.StoreHourlyPredictions(ctx, input); err != nil {
		t.Fatalf("StoreHourlyPredictions: %v", err)
	}
	input[0].HorizonHours = 99

	got, err := repo.GetForecast(ctx, "n1")
	if err != nil {
		t.Fatalf("GetForecast: %v", err)
	}
	if got.Hourly[0].HorizonHours != 1 {
		t.Fatalf("stored slice mutated via input: got horizon %d, want 1", got.Hourly[0].HorizonHours)
	}
	got.Hourly[0].HorizonHours = 77

	gotAgain, err := repo.GetForecast(ctx, "n1")
	if err != nil {
		t.Fatalf("GetForecast second call: %v", err)
	}
	if gotAgain.Hourly[0].HorizonHours != 1 {
		t.Fatalf("stored slice mutated via returned Forecast: got horizon %d, want 1", gotAgain.Hourly[0].HorizonHours)
	}
}

func TestInMemoryPredictions_Concurrency(t *testing.T) {
	ctx := context.Background()
	repo := New(nil, zap.NewNop())
	baseTs := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

	const workers = 12
	const iterations = 50

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			nodeID := fmt.Sprintf("station-%d", workerID%4)
			for i := 0; i < iterations; i++ {
				ts := baseTs.Add(time.Duration(i) * time.Minute)
				temp := float32(15.0 + float32(i)*0.1)

				if _, err := repo.StoreHourlyPredictions(ctx, []HourlyPrediction{
					{NodeID: nodeID, Ts: ts, HorizonHours: 3, TemperatureC: &temp},
					{NodeID: nodeID, Ts: ts, HorizonHours: 1, TemperatureC: &temp},
					{NodeID: nodeID, Ts: ts, HorizonHours: 2, TemperatureC: &temp},
				}); err != nil {
					t.Errorf("StoreHourlyPredictions error: %v", err)
					return
				}

				if _, err := repo.StoreDailyPredictions(ctx, []DailyPrediction{
					{NodeID: nodeID, Ts: ts, HorizonDays: 2, MaxTemperatureC: &temp},
					{NodeID: nodeID, Ts: ts, HorizonDays: 1, MaxTemperatureC: &temp},
				}); err != nil {
					t.Errorf("StoreDailyPredictions error: %v", err)
					return
				}

				forecast, err := repo.GetForecast(ctx, nodeID)
				if err != nil {
					t.Errorf("GetForecast error: %v", err)
					return
				}
				for idx := 1; idx < len(forecast.Hourly); idx++ {
					if forecast.Hourly[idx-1].HorizonHours >= forecast.Hourly[idx].HorizonHours {
						t.Errorf("hourly predictions out of order: %+v", forecast.Hourly)
						return
					}
				}
				for idx := 1; idx < len(forecast.Daily); idx++ {
					if forecast.Daily[idx-1].HorizonDays >= forecast.Daily[idx].HorizonDays {
						t.Errorf("daily predictions out of order: %+v", forecast.Daily)
						return
					}
				}
			}
		}(w)
	}
	wg.Wait()
}

func TestInMemoryPredictions_DeleteNode(t *testing.T) {
	ctx := context.Background()
	repo := New(nil, zap.NewNop())
	ts := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	temp := float32(21.5)

	if _, err := repo.StoreHourlyPredictions(ctx, []HourlyPrediction{
		{NodeID: "station-alpha", Ts: ts, HorizonHours: 1, TemperatureC: &temp},
		{NodeID: "station-beta", Ts: ts, HorizonHours: 1, TemperatureC: &temp},
	}); err != nil {
		t.Fatalf("StoreHourlyPredictions: %v", err)
	}
	if _, err := repo.StoreDailyPredictions(ctx, []DailyPrediction{
		{NodeID: "station-alpha", Ts: ts, HorizonDays: 1, MaxTemperatureC: &temp},
		{NodeID: "station-beta", Ts: ts, HorizonDays: 1, MaxTemperatureC: &temp},
	}); err != nil {
		t.Fatalf("StoreDailyPredictions: %v", err)
	}

	repo.DeleteNode("station-alpha")
	repo.DeleteNode("")

	alphaForecast, err := repo.GetForecast(ctx, "station-alpha")
	if err != nil {
		t.Fatalf("GetForecast(station-alpha): %v", err)
	}
	if len(alphaForecast.Hourly) != 0 || len(alphaForecast.Daily) != 0 {
		t.Errorf("expected station-alpha forecast cleared, got hourly=%d daily=%d", len(alphaForecast.Hourly), len(alphaForecast.Daily))
	}

	betaForecast, err := repo.GetForecast(ctx, "station-beta")
	if err != nil {
		t.Fatalf("GetForecast(station-beta): %v", err)
	}
	if len(betaForecast.Hourly) != 1 || len(betaForecast.Daily) != 1 {
		t.Errorf("expected station-beta forecast preserved, got hourly=%d daily=%d", len(betaForecast.Hourly), len(betaForecast.Daily))
	}
}
