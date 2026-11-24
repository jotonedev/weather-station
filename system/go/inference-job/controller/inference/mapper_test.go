package inference

import (
	"testing"
	"time"

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
					BucketStart:         now,
					AvgTemperatureC:     float32Ptr(21.5),
					AvgHumidityPct:      float32Ptr(55.0),
					AvgPressureHpa:      float32Ptr(1013.2),
					TotalRainfallMm:     float32Ptr(1.8),
					AvgWindSpeedMps:     float32Ptr(4.5),
					AvgWindDirectionDeg: float32Ptr(225.0),
					MaxWindSpeedMps:     float32Ptr(9.0),
					AvgWindUMps:         float32Ptr(3.2),
					AvgWindVMps:         float32Ptr(3.2),
					MaxRainRateMmh:      float32Ptr(4.0),
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
				if obs.GetRainfallMm() != 1.8 {
					t.Fatalf("RainfallMm = %v, want 1.8", obs.GetRainfallMm())
				}
				if obs.GetWindSpeedMps() != 4.5 {
					t.Fatalf("WindSpeedMps = %v, want 4.5", obs.GetWindSpeedMps())
				}
				if obs.GetWindDirectionDeg() != 225.0 {
					t.Fatalf("WindDirectionDeg = %v, want 225.0", obs.GetWindDirectionDeg())
				}
				if obs.GetMaxWindSpeedMps() != 9.0 {
					t.Fatalf("MaxWindSpeedMps = %v, want 9.0", obs.GetMaxWindSpeedMps())
				}
				if obs.GetWindUMps() != 3.2 {
					t.Fatalf("WindUMps = %v, want 3.2", obs.GetWindUMps())
				}
				if obs.GetMaxRainRateMmh() != 4.0 {
					t.Fatalf("MaxRainRateMmh = %v, want 4.0", obs.GetMaxRainRateMmh())
				}
			}
		})
	}
}

func TestBlendTodayPrediction(t *testing.T) {
	tests := []struct {
		name        string
		issueTime   time.Time
		todayActual []*protodatabasev1.ClimateAggregate
		hourlyPreds []*protodatabasev1.HourlyPrediction
		initialProb float32
		wantProbMin float32
		wantProbMax float32
		wantRainMin float32
		wantMinTemp float32
		wantMaxTemp float32
	}{
		{
			name:      "observed rainfall >= 1mm forces rain probability to 100%",
			issueTime: time.Date(2026, 11, 24, 8, 0, 0, 0, time.UTC),
			todayActual: []*protodatabasev1.ClimateAggregate{
				{
					BucketStart:     timestamppb.New(time.Date(2026, 11, 24, 6, 0, 0, 0, time.UTC)),
					MinTemperatureC: float32Ptr(5.0),
					MaxTemperatureC: float32Ptr(8.0),
					TotalRainfallMm: float32Ptr(1.4),
				},
			},
			hourlyPreds: []*protodatabasev1.HourlyPrediction{
				{HorizonHours: 1, TemperatureC: float32Ptr(12.5), RainfallMm: float32Ptr(0.1), RainProbability: float32Ptr(0.2)},
			},
			initialProb: 0.49,
			wantProbMin: 1.0,
			wantProbMax: 1.0,
			wantRainMin: 1.5,
			wantMinTemp: 5.0,
			wantMaxTemp: 12.5,
		},
		{
			name:      "dry morning with heavy afternoon hourly storm boosts today rain chance",
			issueTime: time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC),
			todayActual: []*protodatabasev1.ClimateAggregate{
				{
					BucketStart:     timestamppb.New(time.Date(2026, 11, 24, 3, 0, 0, 0, time.UTC)),
					MinTemperatureC: float32Ptr(4.0),
					MaxTemperatureC: float32Ptr(9.0),
					TotalRainfallMm: float32Ptr(0.0),
				},
			},
			hourlyPreds: []*protodatabasev1.HourlyPrediction{
				{HorizonHours: 5, TemperatureC: float32Ptr(14.0), RainfallMm: float32Ptr(2.2), RainProbability: float32Ptr(0.85)},
			},
			initialProb: 0.35,
			wantProbMin: 0.84,
			wantProbMax: 0.86,
			wantRainMin: 2.2,
			wantMinTemp: 4.0,
			wantMaxTemp: 14.0,
		},
		{
			name:      "late evening dry day caps rain probability by union bound",
			issueTime: time.Date(2026, 11, 24, 22, 0, 0, 0, time.UTC),
			todayActual: []*protodatabasev1.ClimateAggregate{
				{
					BucketStart:     timestamppb.New(time.Date(2026, 11, 24, 14, 0, 0, 0, time.UTC)),
					MinTemperatureC: float32Ptr(3.0),
					MaxTemperatureC: float32Ptr(11.0),
					TotalRainfallMm: float32Ptr(0.0),
				},
			},
			hourlyPreds: []*protodatabasev1.HourlyPrediction{
				{HorizonHours: 1, TemperatureC: float32Ptr(5.0), RainfallMm: float32Ptr(0.0), RainProbability: float32Ptr(0.05)},
				{HorizonHours: 2, TemperatureC: float32Ptr(4.5), RainfallMm: float32Ptr(0.0), RainProbability: float32Ptr(0.05)},
				{HorizonHours: 10, TemperatureC: float32Ptr(12.0), RainfallMm: float32Ptr(3.0), RainProbability: float32Ptr(0.90)},
			},
			initialProb: 0.45,
			wantProbMin: 0.0,
			wantProbMax: 0.11,
			wantRainMin: 0.0,
			wantMinTemp: 3.0,
			wantMaxTemp: 11.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preds := []*protodatabasev1.DailyPrediction{
				{
					NodeId:          "station-alpha",
					HorizonDays:     0,
					MinTemperatureC: float32Ptr(7.0),
					MaxTemperatureC: float32Ptr(10.0),
					RainfallMm:      float32Ptr(0.5),
					RainProbability: float32Ptr(tt.initialProb),
				},
			}
			BlendTodayPrediction(preds, tt.issueTime, tt.todayActual, tt.hourlyPreds)

			today := preds[0]
			if today.GetRainProbability() < tt.wantProbMin || today.GetRainProbability() > tt.wantProbMax {
				t.Fatalf("RainProbability = %v, want in [%v, %v]", today.GetRainProbability(), tt.wantProbMin, tt.wantProbMax)
			}
			if today.GetRainfallMm() < tt.wantRainMin {
				t.Fatalf("RainfallMm = %v, want >= %v", today.GetRainfallMm(), tt.wantRainMin)
			}
			if today.GetMinTemperatureC() != tt.wantMinTemp || today.GetMaxTemperatureC() != tt.wantMaxTemp {
				t.Fatalf("temps = (%v, %v), want (%v, %v)", today.GetMinTemperatureC(), today.GetMaxTemperatureC(), tt.wantMinTemp, tt.wantMaxTemp)
			}
		})
	}
}
