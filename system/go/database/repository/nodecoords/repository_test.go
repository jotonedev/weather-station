package nodecoords

import (
	"context"
	"testing"
	"time"

	duckdb "weather-station/database/client"
	"weather-station/database/repository/prediction"

	"go.uber.org/zap"
)

func TestUpsertEmpty(t *testing.T) {
	repo := &Repository{client: nil, log: zap.NewNop()}
	stored, err := repo.UpsertNodeCoordinates(context.Background(), nil)
	if err != nil {
		t.Fatalf("UpsertNodeCoordinates(empty): %v", err)
	}
	if stored != 0 {
		t.Errorf("rows = %d, want 0", stored)
	}
}

func TestDeleteNodeEmptyID(t *testing.T) {
	repo := &Repository{client: nil, log: zap.NewNop()}
	if err := repo.DeleteNode(context.Background(), ""); err == nil {
		t.Fatal("DeleteNode(\"\") expected error, got nil")
	}
}

func TestDeleteNode_WithoutPredictionTables(t *testing.T) {
	ctx := context.Background()
	dbClient, err := duckdb.NewTestClient(ctx)
	if err != nil {
		t.Fatalf("NewTestClient: %v", err)
	}
	defer dbClient.DB().Close()

	repo := New(dbClient, zap.NewNop())
	_, err = repo.UpsertNodeCoordinates(ctx, []NodeCoordinates{
		{NodeID: "station-alpha", Latitude: 50.1, Longitude: 8.6, ElevationM: 112.0, LocationName: "Alpha"},
		{NodeID: "station-beta", Latitude: 51.2, Longitude: 9.7, ElevationM: 210.0, LocationName: "Beta"},
	})
	if err != nil {
		t.Fatalf("UpsertNodeCoordinates: %v", err)
	}

	ts := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	for _, nodeID := range []string{"station-alpha", "station-beta"} {
		if _, err := dbClient.DB().ExecContext(
			ctx,
			`INSERT INTO weather (node_id, ts, temperature) VALUES (?, ?, ?)`,
			nodeID, ts, 19.5,
		); err != nil {
			t.Fatalf("insert weather row for %s: %v", nodeID, err)
		}
	}

	predRepo := prediction.New(dbClient, zap.NewNop())
	temp := float32(19.5)
	if _, err := predRepo.StoreHourlyPredictions(ctx, []prediction.HourlyPrediction{
		{NodeID: "station-alpha", Ts: ts, HorizonHours: 1, TemperatureC: &temp},
		{NodeID: "station-beta", Ts: ts, HorizonHours: 1, TemperatureC: &temp},
	}); err != nil {
		t.Fatalf("StoreHourlyPredictions: %v", err)
	}
	if _, err := predRepo.StoreDailyPredictions(ctx, []prediction.DailyPrediction{
		{NodeID: "station-alpha", Ts: ts, HorizonDays: 1, MaxTemperatureC: &temp},
		{NodeID: "station-beta", Ts: ts, HorizonDays: 1, MaxTemperatureC: &temp},
	}); err != nil {
		t.Fatalf("StoreDailyPredictions: %v", err)
	}

	if err := repo.DeleteNode(ctx, "station-alpha"); err != nil {
		t.Fatalf("DeleteNode(station-alpha): %v", err)
	}

	alphaForecast, err := predRepo.GetForecast(ctx, "station-alpha")
	if err != nil {
		t.Fatalf("GetForecast(station-alpha): %v", err)
	}
	if len(alphaForecast.Hourly) != 0 || len(alphaForecast.Daily) != 0 {
		t.Errorf("expected empty forecast for deleted node station-alpha, got hourly=%d daily=%d", len(alphaForecast.Hourly), len(alphaForecast.Daily))
	}

	betaForecast, err := predRepo.GetForecast(ctx, "station-beta")
	if err != nil {
		t.Fatalf("GetForecast(station-beta): %v", err)
	}
	if len(betaForecast.Hourly) != 1 || len(betaForecast.Daily) != 1 {
		t.Errorf("expected station-beta forecast to remain, got hourly=%d daily=%d", len(betaForecast.Hourly), len(betaForecast.Daily))
	}

	if _, err := repo.GetNodeCoordinates(ctx, "station-alpha"); err == nil {
		t.Fatalf("expected error getting deleted node station-alpha, got nil")
	}

	var alphaWeatherCount int
	if err := dbClient.DB().QueryRowContext(ctx, `SELECT count(*) FROM weather WHERE node_id = ?`, "station-alpha").Scan(&alphaWeatherCount); err != nil {
		t.Fatalf("query alpha weather count: %v", err)
	}
	if alphaWeatherCount != 0 {
		t.Errorf("alpha weather count = %d, want 0", alphaWeatherCount)
	}

	beta, err := repo.GetNodeCoordinates(ctx, "station-beta")
	if err != nil {
		t.Fatalf("GetNodeCoordinates(station-beta): %v", err)
	}
	if beta.LocationName != "Beta" {
		t.Errorf("beta location = %q, want %q", beta.LocationName, "Beta")
	}

	var betaWeatherCount int
	if err := dbClient.DB().QueryRowContext(ctx, `SELECT count(*) FROM weather WHERE node_id = ?`, "station-beta").Scan(&betaWeatherCount); err != nil {
		t.Fatalf("query beta weather count: %v", err)
	}
	if betaWeatherCount != 1 {
		t.Errorf("beta weather count = %d, want 1", betaWeatherCount)
	}
}

func TestDeleteNode_NonExistentNode(t *testing.T) {
	ctx := context.Background()
	dbClient, err := duckdb.NewTestClient(ctx)
	if err != nil {
		t.Fatalf("NewTestClient: %v", err)
	}
	defer dbClient.DB().Close()

	repo := New(dbClient, zap.NewNop())
	if err := repo.DeleteNode(ctx, "missing-node"); err != nil {
		t.Fatalf("DeleteNode(missing-node): %v", err)
	}
}
