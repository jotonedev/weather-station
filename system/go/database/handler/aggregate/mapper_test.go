package aggregate

import (
	"testing"
	"time"

	"weather-station/common/pb"
	repository "weather-station/database/repository/aggregate"
)

func TestModelClimateToPB(t *testing.T) {
	bucketStart := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	avgTemp := float32(21.5)

	aggregate := repository.ClimateAggregate{
		NodeID:          "n1",
		BucketStart:     bucketStart,
		AvgTemperatureC: &avgTemp,
		SampleCount:     3,
	}

	response := modelClimateToPB(aggregate)
	if response.NodeId != "n1" {
		t.Errorf("NodeId = %q, want n1", response.NodeId)
	}
	if !response.BucketStart.AsTime().Equal(bucketStart) {
		t.Errorf("BucketStart = %v, want %v", response.BucketStart.AsTime(), bucketStart)
	}
	if response.AvgTemperatureC == nil || *response.AvgTemperatureC != avgTemp {
		t.Errorf("AvgTemperatureC = %v, want %v", response.AvgTemperatureC, avgTemp)
	}
	if response.SampleCount != 3 {
		t.Errorf("SampleCount = %d, want 3", response.SampleCount)
	}
	if response.MinTemperatureC != nil {
		t.Errorf("MinTemperatureC = %v, want nil", response.MinTemperatureC)
	}
}

func TestModelAirQualityToPB(t *testing.T) {
	bucketStart := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	avgPm25 := float32(12.0)

	aggregate := repository.AirQualityAggregate{
		NodeID:      "n1",
		BucketStart: bucketStart,
		AvgPm25:     &avgPm25,
		SampleCount: 2,
	}

	response := modelAirQualityToPB(aggregate)
	if response.NodeId != "n1" {
		t.Errorf("NodeId = %q, want n1", response.NodeId)
	}
	if !response.BucketStart.AsTime().Equal(bucketStart) {
		t.Errorf("BucketStart = %v, want %v", response.BucketStart.AsTime(), bucketStart)
	}
	if response.AvgPm25 == nil || *response.AvgPm25 != avgPm25 {
		t.Errorf("AvgPm25 = %v, want %v", response.AvgPm25, avgPm25)
	}
	if response.AvgPm1 != nil {
		t.Errorf("AvgPm1 = %v, want nil", response.AvgPm1)
	}
}

func TestModelClimateToPBTimeHelper(t *testing.T) {
	bucketStart := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	if got := pb.Time(bucketStart); !got.AsTime().Equal(bucketStart) {
		t.Errorf("pb.Time = %v, want %v", got.AsTime(), bucketStart)
	}
}
