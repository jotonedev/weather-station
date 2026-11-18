package weatherstation

import (
	"testing"
	"time"

	"weather-station/common/pb"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func float32Ptr(value float32) *float32 {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
}

func assertPBFloat32Ptr(t *testing.T, fieldName string, got *float32, want float32) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %v", fieldName, want)
	}
	if *got != want {
		t.Fatalf("%s = %v, want %v", fieldName, *got, want)
	}
}

func assertPBInt32Ptr(t *testing.T, fieldName string, got *int32, want int32) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %v", fieldName, want)
	}
	if *got != want {
		t.Fatalf("%s = %v, want %v", fieldName, *got, want)
	}
}

func assertPBFloat32Nil(t *testing.T, fieldName string, got *float32) {
	t.Helper()
	if got != nil {
		t.Fatalf("%s = %v, want nil", fieldName, *got)
	}
}

func assertPBInt32Nil(t *testing.T, fieldName string, got *int32) {
	t.Helper()
	if got != nil {
		t.Fatalf("%s = %v, want nil", fieldName, *got)
	}
}

func TestReadingToPB(t *testing.T) {
	testTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name  string
		input Reading
		check func(t *testing.T, pbReading *protodatabasev1.Reading)
	}{
		{
			name: "converts all populated fields",
			input: Reading{
				NodeID:           "node-1",
				Timestamp:        testTime,
				WindSpeed:        float32Ptr(3.2),
				WindAngle:        float32Ptr(180.5),
				RawWindSpeed:     float32Ptr(3.4),
				RawWindAngle:     float32Ptr(181.0),
				Temperature:      float32Ptr(23.4),
				Humidity:         float32Ptr(56.7),
				Pressure:         float32Ptr(1012.8),
				Rain1h:           float32Ptr(0.4),
				Rain24h:          float32Ptr(4.2),
				RainRate:         float32Ptr(1.1),
				PM1_0:            int32Ptr(8),
				PM2_5:            int32Ptr(15),
				PM10_0:           int32Ptr(24),
				AnemoStatus:      int32Ptr(1),
				AnemoAGC:         int32Ptr(110),
				AnemoAGCVariance: int32Ptr(5),
				VaneStatus:       int32Ptr(2),
				VaneAGC:          int32Ptr(120),
				VaneAGCVariance:  int32Ptr(6),
				RainStatus:       int32Ptr(3),
				RainAGC:          int32Ptr(130),
				RainAGCVariance:  int32Ptr(7),
			},
			check: func(t *testing.T, pbReading *protodatabasev1.Reading) {
				if pbReading.NodeId != "node-1" {
					t.Fatalf("NodeId = %q, want node-1", pbReading.NodeId)
				}
				if !pb.FromTime(pbReading.Timestamp).Equal(testTime) {
					t.Fatalf("Timestamp = %v, want %v", pbReading.Timestamp, testTime)
				}
				assertPBFloat32Ptr(t, "WindSpeed", pbReading.WindSpeed, 3.2)
				assertPBFloat32Ptr(t, "WindAngle", pbReading.WindAngle, 180.5)
				assertPBFloat32Ptr(t, "RawWindSpeed", pbReading.RawWindSpeed, 3.4)
				assertPBFloat32Ptr(t, "RawWindAngle", pbReading.RawWindAngle, 181.0)
				assertPBFloat32Ptr(t, "Temperature", pbReading.Temperature, 23.4)
				assertPBFloat32Ptr(t, "Humidity", pbReading.Humidity, 56.7)
				assertPBFloat32Ptr(t, "Pressure", pbReading.Pressure, 1012.8)
				assertPBFloat32Ptr(t, "Rain1H", pbReading.Rain1H, 0.4)
				assertPBFloat32Ptr(t, "Rain24H", pbReading.Rain24H, 4.2)
				assertPBFloat32Ptr(t, "RainRate", pbReading.RainRate, 1.1)
				assertPBInt32Ptr(t, "Pm1", pbReading.Pm1, 8)
				assertPBInt32Ptr(t, "Pm25", pbReading.Pm25, 15)
				assertPBInt32Ptr(t, "Pm10", pbReading.Pm10, 24)
				assertPBInt32Ptr(t, "AnemoStatus", pbReading.AnemoStatus, 1)
				assertPBInt32Ptr(t, "AnemoAgc", pbReading.AnemoAgc, 110)
				assertPBInt32Ptr(t, "AnemoAgcVariance", pbReading.AnemoAgcVariance, 5)
				assertPBInt32Ptr(t, "VaneStatus", pbReading.VaneStatus, 2)
				assertPBInt32Ptr(t, "VaneAgc", pbReading.VaneAgc, 120)
				assertPBInt32Ptr(t, "VaneAgcVariance", pbReading.VaneAgcVariance, 6)
				assertPBInt32Ptr(t, "RainStatus", pbReading.RainStatus, 3)
				assertPBInt32Ptr(t, "RainAgc", pbReading.RainAgc, 130)
				assertPBInt32Ptr(t, "RainAgcVariance", pbReading.RainAgcVariance, 7)
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
				if !pb.FromTime(pbReading.Timestamp).Equal(testTime) {
					t.Fatalf("Timestamp = %v, want %v", pbReading.Timestamp, testTime)
				}
				assertPBFloat32Nil(t, "WindSpeed", pbReading.WindSpeed)
				assertPBFloat32Nil(t, "WindAngle", pbReading.WindAngle)
				assertPBFloat32Nil(t, "RawWindSpeed", pbReading.RawWindSpeed)
				assertPBFloat32Nil(t, "RawWindAngle", pbReading.RawWindAngle)
				assertPBFloat32Nil(t, "Temperature", pbReading.Temperature)
				assertPBFloat32Nil(t, "Humidity", pbReading.Humidity)
				assertPBFloat32Nil(t, "Pressure", pbReading.Pressure)
				assertPBFloat32Nil(t, "Rain1H", pbReading.Rain1H)
				assertPBFloat32Nil(t, "Rain24H", pbReading.Rain24H)
				assertPBFloat32Nil(t, "RainRate", pbReading.RainRate)
				assertPBInt32Nil(t, "Pm1", pbReading.Pm1)
				assertPBInt32Nil(t, "Pm25", pbReading.Pm25)
				assertPBInt32Nil(t, "Pm10", pbReading.Pm10)
				assertPBInt32Nil(t, "AnemoStatus", pbReading.AnemoStatus)
				assertPBInt32Nil(t, "AnemoAgc", pbReading.AnemoAgc)
				assertPBInt32Nil(t, "AnemoAgcVariance", pbReading.AnemoAgcVariance)
				assertPBInt32Nil(t, "VaneStatus", pbReading.VaneStatus)
				assertPBInt32Nil(t, "VaneAgc", pbReading.VaneAgc)
				assertPBInt32Nil(t, "VaneAgcVariance", pbReading.VaneAgcVariance)
				assertPBInt32Nil(t, "RainStatus", pbReading.RainStatus)
				assertPBInt32Nil(t, "RainAgc", pbReading.RainAgc)
				assertPBInt32Nil(t, "RainAgcVariance", pbReading.RainAgcVariance)
			},
		},
		{
			name: "preserves explicit zero values on optional pointers",
			input: Reading{
				NodeID:      "node-3",
				Timestamp:   testTime,
				Temperature: float32Ptr(0),
				Rain1h:      float32Ptr(0),
				PM1_0:       int32Ptr(0),
				AnemoStatus: int32Ptr(0),
			},
			check: func(t *testing.T, pbReading *protodatabasev1.Reading) {
				assertPBFloat32Ptr(t, "Temperature", pbReading.Temperature, 0)
				assertPBFloat32Ptr(t, "Rain1H", pbReading.Rain1H, 0)
				assertPBInt32Ptr(t, "Pm1", pbReading.Pm1, 0)
				assertPBInt32Ptr(t, "AnemoStatus", pbReading.AnemoStatus, 0)
				assertPBFloat32Nil(t, "WindSpeed", pbReading.WindSpeed)
				assertPBInt32Nil(t, "Pm25", pbReading.Pm25)
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
