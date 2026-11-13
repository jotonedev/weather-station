package aggregate

import (
	"context"
	"math"
	"testing"
	"time"

	duckdb "weather-station/database/client"

	"go.uber.org/zap"
)

func insertWeatherSample(t *testing.T, client *duckdb.Client, nodeID string, ts time.Time, temp float64, rain1h, rainRate *float64) {
	t.Helper()
	_, err := client.DB().ExecContext(
		context.Background(),
		`INSERT INTO weather (node_id, ts, temperature, humidity, pressure, rain_1h, rain_rate, wind_speed, wind_angle)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nodeID, ts.UTC(), temp, 60.0, 1013.25, rain1h, rainRate, 3.0, 180.0,
	)
	if err != nil {
		t.Fatalf("insert weather sample: %v", err)
	}
}

func ptrFloat64(v float64) *float64 {
	return &v
}

func TestGetClimateAggregates_HourlyLateRainAndNullHandling(t *testing.T) {
	ctx := context.Background()
	client, err := duckdb.NewTestClient(ctx)
	if err != nil {
		t.Fatalf("NewTestClient: %v", err)
	}
	defer client.DB().Close()

	repo := New(client, zap.NewNop())
	base := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)

	// Hour 10:00-11:00: rain falls late in the hour; arg_max should pick 4.2, not average 1.4.
	insertWeatherSample(t, client, "node-1", base.Add(10*time.Minute), 15.0, ptrFloat64(0.0), nil)
	insertWeatherSample(t, client, "node-1", base.Add(30*time.Minute), 15.0, ptrFloat64(0.0), nil)
	insertWeatherSample(t, client, "node-1", base.Add(55*time.Minute), 14.5, ptrFloat64(4.2), nil)

	// Hour 11:00-12:00: latest sample has NULL rain_1h; earlier sample has 1.8.
	insertWeatherSample(t, client, "node-1", base.Add(70*time.Minute), 14.0, ptrFloat64(1.8), nil)
	insertWeatherSample(t, client, "node-1", base.Add(110*time.Minute), 14.0, nil, nil)

	// Hour 12:00-13:00: all samples have NULL rain_1h; TotalRainfallMm should be nil.
	insertWeatherSample(t, client, "node-1", base.Add(130*time.Minute), 16.0, nil, nil)

	aggs, err := repo.GetClimateAggregates(ctx, "node-1", base, base.Add(3*time.Hour), 3600)
	if err != nil {
		t.Fatalf("GetClimateAggregates: %v", err)
	}
	if len(aggs) != 3 {
		t.Fatalf("len(aggs) = %d, want 3", len(aggs))
	}

	if aggs[0].TotalRainfallMm == nil || math.Abs(float64(*aggs[0].TotalRainfallMm)-4.2) > 1e-4 {
		t.Fatalf("hour 0 TotalRainfallMm = %v, want 4.2", aggs[0].TotalRainfallMm)
	}
	if aggs[1].TotalRainfallMm == nil || math.Abs(float64(*aggs[1].TotalRainfallMm)-1.8) > 1e-4 {
		t.Fatalf("hour 1 TotalRainfallMm = %v, want 1.8", aggs[1].TotalRainfallMm)
	}
	if aggs[2].TotalRainfallMm != nil {
		t.Fatalf("hour 2 TotalRainfallMm = %v, want nil", *aggs[2].TotalRainfallMm)
	}
}

func TestGetClimateAggregates_DailyMultiHourStorm(t *testing.T) {
	ctx := context.Background()
	client, err := duckdb.NewTestClient(ctx)
	if err != nil {
		t.Fatalf("NewTestClient: %v", err)
	}
	defer client.DB().Close()

	repo := New(client, zap.NewNop())
	dayStart := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)

	// Hour 01:00: 0.5 -> 2.5 mm
	insertWeatherSample(t, client, "node-1", dayStart.Add(1*time.Hour+15*time.Minute), 12.0, ptrFloat64(0.5), nil)
	insertWeatherSample(t, client, "node-1", dayStart.Add(1*time.Hour+50*time.Minute), 12.0, ptrFloat64(2.5), nil)

	// Hour 02:00: 3.0 -> 1.0 mm (end-of-hour is 1.0)
	insertWeatherSample(t, client, "node-1", dayStart.Add(2*time.Hour+10*time.Minute), 11.5, ptrFloat64(3.0), nil)
	insertWeatherSample(t, client, "node-1", dayStart.Add(2*time.Hour+55*time.Minute), 11.5, ptrFloat64(1.0), nil)

	// Hour 14:00: NULL rain_1h
	insertWeatherSample(t, client, "node-1", dayStart.Add(14*time.Hour), 15.0, nil, nil)

	aggs, err := repo.GetClimateAggregates(ctx, "node-1", dayStart, dayStart.Add(24*time.Hour), 86400)
	if err != nil {
		t.Fatalf("GetClimateAggregates: %v", err)
	}
	if len(aggs) != 1 {
		t.Fatalf("len(aggs) = %d, want 1", len(aggs))
	}

	// Expected daily total: 2.5 + 1.0 = 3.5 mm
	if aggs[0].TotalRainfallMm == nil || math.Abs(float64(*aggs[0].TotalRainfallMm)-3.5) > 1e-4 {
		t.Fatalf("daily TotalRainfallMm = %v, want 3.5", aggs[0].TotalRainfallMm)
	}
	if aggs[0].SampleCount != 5 {
		t.Fatalf("daily SampleCount = %d, want 5", aggs[0].SampleCount)
	}
}

func TestGetClimateAggregates_SubHourlyFallback(t *testing.T) {
	ctx := context.Background()
	client, err := duckdb.NewTestClient(ctx)
	if err != nil {
		t.Fatalf("NewTestClient: %v", err)
	}
	defer client.DB().Close()

	repo := New(client, zap.NewNop())
	base := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)

	// Bucket 0 (08:00-08:15): rain_rate = 4.0 mm/h -> 4.0 * (900/3600) = 1.0 mm
	insertWeatherSample(t, client, "node-1", base.Add(5*time.Minute), 10.0, ptrFloat64(2.0), ptrFloat64(4.0))

	// Bucket 1 (08:15-08:30): rain_rate is NULL, rain_1h = 8.0 -> 8.0 * (900/3600) = 2.0 mm
	insertWeatherSample(t, client, "node-1", base.Add(20*time.Minute), 10.0, ptrFloat64(8.0), nil)

	aggs, err := repo.GetClimateAggregates(ctx, "node-1", base, base.Add(30*time.Minute), 900)
	if err != nil {
		t.Fatalf("GetClimateAggregates: %v", err)
	}
	if len(aggs) != 2 {
		t.Fatalf("len(aggs) = %d, want 2", len(aggs))
	}
	if aggs[0].TotalRainfallMm == nil || math.Abs(float64(*aggs[0].TotalRainfallMm)-1.0) > 1e-4 {
		t.Fatalf("bucket 0 TotalRainfallMm = %v, want 1.0", aggs[0].TotalRainfallMm)
	}
	if aggs[1].TotalRainfallMm == nil || math.Abs(float64(*aggs[1].TotalRainfallMm)-2.0) > 1e-4 {
		t.Fatalf("bucket 1 TotalRainfallMm = %v, want 2.0", aggs[1].TotalRainfallMm)
	}
}
