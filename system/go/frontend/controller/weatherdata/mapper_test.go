package weatherdata

import (
	"testing"
	"time"

	"weather-station/common/pb"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestPBPredictionToView(t *testing.T) {
	issueTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	temp := float32(18.4)
	rain := float32(1.2)
	confRain := float32(0.9)

	testCases := []struct {
		name       string
		prediction *protodatabasev1.Prediction
		check      func(t *testing.T, view ForecastItemView)
	}{
		{
			name: "converts hourly forecast prediction with light rain and confidence",
			prediction: &protodatabasev1.Prediction{
				NodeId:             "station-1",
				IssueTime:          pb.Time(issueTime),
				HorizonHours:       3,
				TemperatureC:       &temp,
				RainfallMm:         &rain,
				RainfallConfidence: &confRain,
			},
			check: func(t *testing.T, view ForecastItemView) {
				if view.HorizonLabel != "15:04 UTC" && view.HorizonLabel != "15:00 UTC" {
					t.Fatalf("HorizonLabel = %q, want 15:00 UTC", view.HorizonLabel)
				}
				if view.TargetTimeISO != "2026-10-10T15:00:00Z" {
					t.Fatalf("TargetTimeISO = %q, want 2026-10-10T15:00:00Z", view.TargetTimeISO)
				}
				if view.IsDaily {
					t.Fatal("expected IsDaily to be false for 3h horizon")
				}
				if view.TimeSubLabel != "+3h" {
					t.Fatalf("TimeSubLabel = %q, want +3h", view.TimeSubLabel)
				}
				if view.ConditionIcon != "wi-showers" {
					t.Fatalf("ConditionIcon = %q, want wi-showers", view.ConditionIcon)
				}
				if view.Temperature != "18.4 °C" {
					t.Fatalf("Temperature = %q, want 18.4 °C", view.Temperature)
				}
				if view.Rainfall != "1.2 mm" {
					t.Fatalf("Rainfall = %q, want 1.2 mm", view.Rainfall)
				}
				if view.Confidence != "90%" || view.ConfidencePct != 90 {
					t.Fatalf("Confidence = %q (%d), want 90%% (90)", view.Confidence, view.ConfidencePct)
				}
			},
		},
		{
			name: "converts multi-day horizon label with min and max temperature",
			prediction: &protodatabasev1.Prediction{
				NodeId:          "station-1",
				IssueTime:       pb.Time(issueTime),
				HorizonHours:    48,
				MinTemperatureC: &temp,
				MaxTemperatureC: &temp,
			},
			check: func(t *testing.T, view ForecastItemView) {
				if view.HorizonLabel != "Mon 12 Oct" {
					t.Fatalf("HorizonLabel = %q, want Mon 12 Oct", view.HorizonLabel)
				}
				if view.TargetTimeISO != "2026-10-12T12:00:00Z" {
					t.Fatalf("TargetTimeISO = %q, want 2026-10-12T12:00:00Z", view.TargetTimeISO)
				}
				if !view.IsDaily {
					t.Fatal("expected IsDaily to be true for 48h horizon")
				}
				if view.TimeSubLabel != "+2d" {
					t.Fatalf("TimeSubLabel = %q, want +2d", view.TimeSubLabel)
				}
				if view.MinTemperature != "18.4 °C" || view.MaxTemperature != "18.4 °" {
					t.Fatalf("Min/Max = %q / %q, want 18.4 °C / 18.4 °", view.MinTemperature, view.MaxTemperature)
				}
				if view.ConditionIcon != "wi-day-cloudy" {
					t.Fatalf("ConditionIcon = %q, want wi-day-cloudy", view.ConditionIcon)
				}
			},
		},
		{
			name:       "handles nil prediction",
			prediction: nil,
			check: func(t *testing.T, view ForecastItemView) {
				if view.HorizonHours != 0 {
					t.Fatalf("HorizonHours = %d, want 0", view.HorizonHours)
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			view := pbPredictionToView(testCase.prediction)
			testCase.check(t, view)
		})
	}
}

func TestClassifyAirQuality(t *testing.T) {
	val := func(n int32) *int32 { return &n }

	tests := []struct {
		name      string
		pm25      *int32
		pm10      *int32
		wantLabel string
		wantLevel string
	}{
		{"both nil", nil, nil, "No Data", "unknown"},
		{"pm25 good", val(3), nil, "Good", "good"},
		{"pm25 fair", val(10), nil, "Fair", "fair"},
		{"pm25 moderate", val(30), nil, "Moderate", "moderate"},
		{"pm25 poor", val(70), nil, "Poor", "poor"},
		{"pm25 very poor", val(110), nil, "Very Poor", "very-poor"},
		{"pm25 extremely poor", val(160), nil, "Extremely Poor", "extremely-poor"},
		{"pm10 good", nil, val(12), "Good", "good"},
		{"pm10 fair", nil, val(35), "Fair", "fair"},
		{"pm10 moderate", nil, val(80), "Moderate", "moderate"},
		{"pm10 poor", nil, val(150), "Poor", "poor"},
		{"pm10 very poor", nil, val(220), "Very Poor", "very-poor"},
		{"pm10 extremely poor", nil, val(300), "Extremely Poor", "extremely-poor"},
		{"worst of chooses higher severity (pm25 good, pm10 poor)", val(4), val(160), "Poor", "poor"},
		{"worst of chooses higher severity (pm25 moderate, pm10 fair)", val(25), val(20), "Moderate", "moderate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label, level := classifyAirQuality(tt.pm25, tt.pm10)
			if label != tt.wantLabel || level != tt.wantLevel {
				t.Errorf("classifyAirQuality(%v, %v) = (%q, %q), want (%q, %q)", tt.pm25, tt.pm10, label, level, tt.wantLabel, tt.wantLevel)
			}
		})
	}
}
