package weatherdata

import (
	"testing"
	"time"

	"weather-station/common/pb"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestPBReadingToView(t *testing.T) {
	testTime := time.Date(2026, 10, 10, 14, 30, 0, 0, time.UTC)
	temp := float32(22.5)
	freezingTemp := float32(-2.0)
	humidity := float32(60.0)
	highHumidity := float32(90.0)
	pressure := float32(1012.3)
	windSpeed := float32(5.1)
	windAngle := float32(315.0)
	rainRate := float32(5.2)
	lightRain := float32(1.0)
	pm25 := int32(15)
	goodPm25 := int32(8)

	testCases := []struct {
		name    string
		reading *protodatabasev1.Reading
		check   func(t *testing.T, view ReadingView)
	}{
		{
			name: "converts populated fields with correct units and derived indicators",
			reading: &protodatabasev1.Reading{
				NodeId:      "station-1",
				Timestamp:   pb.Time(testTime),
				Temperature: &temp,
				Humidity:    &humidity,
				Pressure:    &pressure,
				WindSpeed:   &windSpeed,
				WindAngle:   &windAngle,
				Pm25:        &pm25,
			},
			check: func(t *testing.T, view ReadingView) {
				if view.NodeID != "station-1" {
					t.Fatalf("NodeID = %q, want station-1", view.NodeID)
				}
				if view.Temperature != "22.5 °C" {
					t.Fatalf("Temperature = %q, want 22.5 °C", view.Temperature)
				}
				if view.Humidity != "60.0 %" {
					t.Fatalf("Humidity = %q, want 60.0 %%", view.Humidity)
				}
				if view.HumidityPercent != 60 {
					t.Fatalf("HumidityPercent = %d, want 60", view.HumidityPercent)
				}
				if view.Pressure != "1012.3 hPa" {
					t.Fatalf("Pressure = %q, want 1012.3 hPa", view.Pressure)
				}
				if view.WindSpeed != "5.1 m/s" {
					t.Fatalf("WindSpeed = %q, want 5.1 m/s", view.WindSpeed)
				}
				if view.WindCardinal != "NW" || !view.HasWindAngle || view.WindRotationDeg != 315 {
					t.Fatalf("Wind = %q (%d, %v), want NW (315, true)", view.WindCardinal, view.WindRotationDeg, view.HasWindAngle)
				}
				if view.PM2_5 != "15 µg/m³" {
					t.Fatalf("PM2_5 = %q, want 15 µg/m³", view.PM2_5)
				}
				if view.AirQualityLabel != "Moderate" || view.AirQualityLevel != "moderate" {
					t.Fatalf("AirQuality = %q/%q, want Moderate/moderate", view.AirQualityLabel, view.AirQualityLevel)
				}
				if view.ConditionIcon != "wi-day-cloudy" {
					t.Fatalf("ConditionIcon = %q, want wi-day-cloudy", view.ConditionIcon)
				}
				if view.VOC != "—" {
					t.Fatalf("VOC = %q, want —", view.VOC)
				}
			},
		},
		{
			name: "classifies heavy rain and good air quality",
			reading: &protodatabasev1.Reading{
				NodeId:      "station-2",
				Temperature: &temp,
				Humidity:    &highHumidity,
				RainRate:    &rainRate,
				Pm25:        &goodPm25,
			},
			check: func(t *testing.T, view ReadingView) {
				if view.ConditionIcon != "wi-rain" || view.ConditionLabel != "Heavy Rain" {
					t.Fatalf("Condition = %q/%q, want wi-rain/Heavy Rain", view.ConditionIcon, view.ConditionLabel)
				}
				if view.AirQualityLabel != "Good" || view.AirQualityLevel != "good" {
					t.Fatalf("AirQuality = %q/%q, want Good/good", view.AirQualityLabel, view.AirQualityLevel)
				}
			},
		},
		{
			name: "classifies snow when precipitation occurs at freezing temperature",
			reading: &protodatabasev1.Reading{
				NodeId:      "station-3",
				Temperature: &freezingTemp,
				RainRate:    &lightRain,
			},
			check: func(t *testing.T, view ReadingView) {
				if view.ConditionIcon != "wi-snow" || view.ConditionLabel != "Snow" {
					t.Fatalf("Condition = %q/%q, want wi-snow/Snow", view.ConditionIcon, view.ConditionLabel)
				}
			},
		},
		{
			name:    "handles nil reading",
			reading: nil,
			check: func(t *testing.T, view ReadingView) {
				if view.NodeID != "" {
					t.Fatalf("NodeID = %q, want empty", view.NodeID)
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			view := pbReadingToView(testCase.reading)
			testCase.check(t, view)
		})
	}
}

func TestPBPredictionToView(t *testing.T) {
	issueTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	temp := float32(18.4)
	humidity := float32(72.0)
	rain := float32(1.2)
	windDir := float32(90.0)
	confTemp := float32(0.8)
	confRain := float32(0.9)

	testCases := []struct {
		name       string
		prediction *protodatabasev1.Prediction
		check      func(t *testing.T, view ForecastItemView)
	}{
		{
			name: "converts hourly forecast prediction with light rain and confidence",
			prediction: &protodatabasev1.Prediction{
				NodeId:                "station-1",
				IssueTime:             pb.Time(issueTime),
				HorizonHours:          3,
				TemperatureC:          &temp,
				HumidityPct:           &humidity,
				RainfallMm:            &rain,
				WindDirectionDeg:      &windDir,
				TemperatureConfidence: &confTemp,
				RainfallConfidence:    &confRain,
			},
			check: func(t *testing.T, view ForecastItemView) {
				if view.HorizonLabel != "15:04 UTC" && view.HorizonLabel != "15:00 UTC" {
					t.Fatalf("HorizonLabel = %q, want 15:00 UTC", view.HorizonLabel)
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
				if view.WindCardinal != "E" || !view.HasWindDirection {
					t.Fatalf("WindCardinal = %q (%v), want E (true)", view.WindCardinal, view.HasWindDirection)
				}
				if view.Confidence != "85%" || view.ConfidencePct != 85 {
					t.Fatalf("Confidence = %q (%d), want 85%% (85)", view.Confidence, view.ConfidencePct)
				}
			},
		},
		{
			name: "converts multi-day horizon label",
			prediction: &protodatabasev1.Prediction{
				NodeId:       "station-1",
				IssueTime:    pb.Time(issueTime),
				HorizonHours: 48,
				TemperatureC: &temp,
			},
			check: func(t *testing.T, view ForecastItemView) {
				if view.HorizonLabel != "Mon 12 Oct" {
					t.Fatalf("HorizonLabel = %q, want Mon 12 Oct", view.HorizonLabel)
				}
				if view.TimeSubLabel != "+2d" {
					t.Fatalf("TimeSubLabel = %q, want +2d", view.TimeSubLabel)
				}
				if view.ConditionIcon != "wi-day-sunny" {
					t.Fatalf("ConditionIcon = %q, want wi-day-sunny", view.ConditionIcon)
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
