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
	humidity := float32(60.0)
	pressure := float32(1012.3)
	windSpeed := float32(5.1)
	pm25 := int32(15)

	testCases := []struct {
		name    string
		reading *protodatabasev1.Reading
		check   func(t *testing.T, view ReadingView)
	}{
		{
			name: "converts populated fields with correct units",
			reading: &protodatabasev1.Reading{
				NodeId:      "station-1",
				Timestamp:   pb.Time(testTime),
				Temperature: &temp,
				Humidity:    &humidity,
				Pressure:    &pressure,
				WindSpeed:   &windSpeed,
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
				if view.Pressure != "1012.3 hPa" {
					t.Fatalf("Pressure = %q, want 1012.3 hPa", view.Pressure)
				}
				if view.WindSpeed != "5.1 m/s" {
					t.Fatalf("WindSpeed = %q, want 5.1 m/s", view.WindSpeed)
				}
				if view.PM2_5 != "15 µg/m³" {
					t.Fatalf("PM2_5 = %q, want 15 µg/m³", view.PM2_5)
				}
				if view.VOC != "—" {
					t.Fatalf("VOC = %q, want —", view.VOC)
				}
			},
		},
		{
			name: "handles nil reading",
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
