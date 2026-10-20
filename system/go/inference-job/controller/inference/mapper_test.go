package inference

import (
	"testing"

	"google.golang.org/protobuf/types/known/timestamppb"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func float32Ptr(value float32) *float32 {
	return &value
}

func TestToTimeSeriesObservations(t *testing.T) {
	now := timestamppb.Now()

	tests := []struct {
		name       string
		aggregates []*protodatabasev1.ClimateAggregate
		wantCount  int
	}{
		{
			name:       "nil slice returns nil",
			aggregates: nil,
			wantCount:  0,
		},
		{
			name:       "empty slice returns nil",
			aggregates: []*protodatabasev1.ClimateAggregate{},
			wantCount:  0,
		},
		{
			name: "skips nil aggregate pointers",
			aggregates: []*protodatabasev1.ClimateAggregate{
				nil,
				{
					BucketStart:      now,
					AvgTemperatureC:  float32Ptr(21.5),
					AvgHumidityPct:   float32Ptr(55.0),
					AvgPressureHpa:   float32Ptr(1013.2),
				},
				nil,
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observations := ToTimeSeriesObservations(tt.aggregates)
			if len(observations) != tt.wantCount {
				t.Fatalf("ToTimeSeriesObservations() length = %d, want %d", len(observations), tt.wantCount)
			}
			if tt.wantCount > 0 {
				obs := observations[0]
				if obs.GetTemperatureC() != 21.5 {
					t.Fatalf("TemperatureC = %v, want 21.5", obs.GetTemperatureC())
				}
				if obs.GetHumidityPct() != 55.0 {
					t.Fatalf("HumidityPct = %v, want 55.0", obs.GetHumidityPct())
				}
				if obs.GetPressureHpa() != 1013.2 {
					t.Fatalf("PressureHpa = %v, want 1013.2", obs.GetPressureHpa())
				}
			}
		})
	}
}
