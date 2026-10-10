package weatherstation

import (
	"testing"
	"time"

	"weather-station/common/pb"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestReadingToPB(t *testing.T) {
	testTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	temperature := float32(23.4)
	humidity := float32(56.7)
	windSpeed := float32(3.2)
	pm25 := int32(15)

	testCases := []struct {
		name  string
		input Reading
		check func(t *testing.T, pbReading *protodatabasev1.Reading)
	}{
		{
			name: "converts all populated fields",
			input: Reading{
				NodeID:      "node-1",
				Timestamp:   testTime,
				Temperature: &temperature,
				Humidity:    &humidity,
				WindSpeed:   &windSpeed,
				PM2_5:       &pm25,
			},
			check: func(t *testing.T, pbReading *protodatabasev1.Reading) {
				if pbReading.NodeId != "node-1" {
					t.Fatalf("NodeId = %q, want node-1", pbReading.NodeId)
				}
				if !pb.FromTime(pbReading.Timestamp).Equal(testTime) {
					t.Fatalf("Timestamp = %v, want %v", pbReading.Timestamp, testTime)
				}
				if pbReading.Temperature == nil || *pbReading.Temperature != temperature {
					t.Fatalf("Temperature = %v, want %v", pbReading.Temperature, temperature)
				}
				if pbReading.Humidity == nil || *pbReading.Humidity != humidity {
					t.Fatalf("Humidity = %v, want %v", pbReading.Humidity, humidity)
				}
				if pbReading.WindSpeed == nil || *pbReading.WindSpeed != windSpeed {
					t.Fatalf("WindSpeed = %v, want %v", pbReading.WindSpeed, windSpeed)
				}
				if pbReading.Pm25 == nil || *pbReading.Pm25 != pm25 {
					t.Fatalf("Pm25 = %v, want %v", pbReading.Pm25, pm25)
				}
				if pbReading.Pressure != nil {
					t.Fatalf("Pressure = %v, want nil", pbReading.Pressure)
				}
			},
		},
		{
			name: "converts empty optional fields to nil",
			input: Reading{
				NodeID:    "node-2",
				Timestamp: testTime,
			},
			check: func(t *testing.T, pbReading *protodatabasev1.Reading) {
				if pbReading.NodeId != "node-2" {
					t.Fatalf("NodeId = %q, want node-2", pbReading.NodeId)
				}
				if pbReading.Temperature != nil {
					t.Fatalf("Temperature = %v, want nil", pbReading.Temperature)
				}
				if pbReading.WindSpeed != nil {
					t.Fatalf("WindSpeed = %v, want nil", pbReading.WindSpeed)
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := readingToPB(testCase.input)
			testCase.check(t, result)
		})
	}
}
