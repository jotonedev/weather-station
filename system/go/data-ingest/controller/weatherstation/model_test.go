package weatherstation

import (
	"encoding/json"
	"testing"
	"time"
)

func assertReadingFloat32Ptr(t *testing.T, fieldName string, got *float32, want float32) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %v", fieldName, want)
	}
	if *got != want {
		t.Fatalf("%s = %v, want %v", fieldName, *got, want)
	}
}

func assertReadingInt32Ptr(t *testing.T, fieldName string, got *int32, want int32) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %v", fieldName, want)
	}
	if *got != want {
		t.Fatalf("%s = %v, want %v", fieldName, *got, want)
	}
}

func assertReadingFloat32Nil(t *testing.T, fieldName string, got *float32) {
	t.Helper()
	if got != nil {
		t.Fatalf("%s = %v, want nil", fieldName, *got)
	}
}

func assertReadingInt32Nil(t *testing.T, fieldName string, got *int32) {
	t.Helper()
	if got != nil {
		t.Fatalf("%s = %v, want nil", fieldName, *got)
	}
}

func TestReadingUnmarshalJSON(t *testing.T) {
	primaryTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	aliasTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name    string
		payload string
		wantErr bool
		check   func(t *testing.T, reading Reading)
	}{
		{
			name: "primary keys populate all 24 fields",
			payload: `{
				"node_id": "station-1",
				"timestamp": "2026-10-10T12:00:00Z",
				"wind_speed": 1.5,
				"wind_angle": 2.5,
				"raw_wind_speed": 3.5,
				"raw_wind_angle": 4.5,
				"temperature": 21.5,
				"humidity": 65.5,
				"pressure": 1013.25,
				"rain_1h": 0.5,
				"rain_24h": 5.5,
				"rain_rate": 1.25,
				"pm1_0": 10,
				"pm2_5": 20,
				"pm10_0": 30,
				"anemo_status": 1,
				"anemo_agc": 101,
				"anemo_agc_variance": 11,
				"vane_status": 2,
				"vane_agc": 102,
				"vane_agc_variance": 12,
				"rain_status": 3,
				"rain_agc": 103,
				"rain_agc_variance": 13
			}`,
			check: func(t *testing.T, reading Reading) {
				if reading.NodeID != "station-1" {
					t.Fatalf("NodeID = %q, want station-1", reading.NodeID)
				}
				if !reading.Timestamp.Equal(primaryTime) {
					t.Fatalf("Timestamp = %v, want %v", reading.Timestamp, primaryTime)
				}
				assertReadingFloat32Ptr(t, "WindSpeed", reading.WindSpeed, 1.5)
				assertReadingFloat32Ptr(t, "WindAngle", reading.WindAngle, 2.5)
				assertReadingFloat32Ptr(t, "RawWindSpeed", reading.RawWindSpeed, 3.5)
				assertReadingFloat32Ptr(t, "RawWindAngle", reading.RawWindAngle, 4.5)
				assertReadingFloat32Ptr(t, "Temperature", reading.Temperature, 21.5)
				assertReadingFloat32Ptr(t, "Humidity", reading.Humidity, 65.5)
				assertReadingFloat32Ptr(t, "Pressure", reading.Pressure, 1013.25)
				assertReadingFloat32Ptr(t, "Rain1h", reading.Rain1h, 0.5)
				assertReadingFloat32Ptr(t, "Rain24h", reading.Rain24h, 5.5)
				assertReadingFloat32Ptr(t, "RainRate", reading.RainRate, 1.25)
				assertReadingInt32Ptr(t, "PM1_0", reading.PM1_0, 10)
				assertReadingInt32Ptr(t, "PM2_5", reading.PM2_5, 20)
				assertReadingInt32Ptr(t, "PM10_0", reading.PM10_0, 30)
				assertReadingInt32Ptr(t, "AnemoStatus", reading.AnemoStatus, 1)
				assertReadingInt32Ptr(t, "AnemoAGC", reading.AnemoAGC, 101)
				assertReadingInt32Ptr(t, "AnemoAGCVariance", reading.AnemoAGCVariance, 11)
				assertReadingInt32Ptr(t, "VaneStatus", reading.VaneStatus, 2)
				assertReadingInt32Ptr(t, "VaneAGC", reading.VaneAGC, 102)
				assertReadingInt32Ptr(t, "VaneAGCVariance", reading.VaneAGCVariance, 12)
				assertReadingInt32Ptr(t, "RainStatus", reading.RainStatus, 3)
				assertReadingInt32Ptr(t, "RainAGC", reading.RainAGC, 103)
				assertReadingInt32Ptr(t, "RainAGCVariance", reading.RainAGCVariance, 13)
			},
		},
		{
			name:    "alias keys populate fallback fields",
			payload: `{"node_id":"station-alias","ts":"2025-01-01T00:00:00Z","rain1h":1.25,"rain24h":4.5,"pm1":7,"pm25":14,"pm10":21}`,
			check: func(t *testing.T, reading Reading) {
				if reading.NodeID != "station-alias" {
					t.Fatalf("NodeID = %q, want station-alias", reading.NodeID)
				}
				if !reading.Timestamp.Equal(aliasTime) {
					t.Fatalf("Timestamp = %v, want %v", reading.Timestamp, aliasTime)
				}
				assertReadingFloat32Ptr(t, "Rain1h", reading.Rain1h, 1.25)
				assertReadingFloat32Ptr(t, "Rain24h", reading.Rain24h, 4.5)
				assertReadingInt32Ptr(t, "PM1_0", reading.PM1_0, 7)
				assertReadingInt32Ptr(t, "PM2_5", reading.PM2_5, 14)
				assertReadingInt32Ptr(t, "PM10_0", reading.PM10_0, 21)
				assertReadingFloat32Nil(t, "Temperature", reading.Temperature)
				assertReadingInt32Nil(t, "AnemoStatus", reading.AnemoStatus)
			},
		},
		{
			name: "primary keys including zero values take precedence over alias keys",
			payload: `{
				"node_id": "station-precedence",
				"timestamp": "2026-10-10T12:00:00Z",
				"ts": "2025-01-01T00:00:00Z",
				"rain_1h": 0,
				"rain1h": 4.5,
				"rain_24h": 0,
				"rain24h": 12.0,
				"pm1_0": 0,
				"pm1": 99,
				"pm2_5": 0,
				"pm25": 88,
				"pm10_0": 0,
				"pm10": 77
			}`,
			check: func(t *testing.T, reading Reading) {
				if !reading.Timestamp.Equal(primaryTime) {
					t.Fatalf("Timestamp = %v, want %v", reading.Timestamp, primaryTime)
				}
				assertReadingFloat32Ptr(t, "Rain1h", reading.Rain1h, 0)
				assertReadingFloat32Ptr(t, "Rain24h", reading.Rain24h, 0)
				assertReadingInt32Ptr(t, "PM1_0", reading.PM1_0, 0)
				assertReadingInt32Ptr(t, "PM2_5", reading.PM2_5, 0)
				assertReadingInt32Ptr(t, "PM10_0", reading.PM10_0, 0)
			},
		},
		{
			name: "null primary keys fall back to alias keys",
			payload: `{
				"node_id": "station-null-fallback",
				"timestamp": null,
				"ts": "2026-10-10T12:00:00Z",
				"rain_1h": null,
				"rain1h": 1.25,
				"rain_24h": null,
				"rain24h": 3.5,
				"pm1_0": null,
				"pm1": 9,
				"pm2_5": null,
				"pm25": 18,
				"pm10_0": null,
				"pm10": 27
			}`,
			check: func(t *testing.T, reading Reading) {
				if !reading.Timestamp.Equal(primaryTime) {
					t.Fatalf("Timestamp = %v, want %v", reading.Timestamp, primaryTime)
				}
				assertReadingFloat32Ptr(t, "Rain1h", reading.Rain1h, 1.25)
				assertReadingFloat32Ptr(t, "Rain24h", reading.Rain24h, 3.5)
				assertReadingInt32Ptr(t, "PM1_0", reading.PM1_0, 9)
				assertReadingInt32Ptr(t, "PM2_5", reading.PM2_5, 18)
				assertReadingInt32Ptr(t, "PM10_0", reading.PM10_0, 27)
			},
		},
		{
			name:    "empty json leaves all fields zero or nil",
			payload: `{}`,
			check: func(t *testing.T, reading Reading) {
				if reading.NodeID != "" {
					t.Fatalf("NodeID = %q, want empty string", reading.NodeID)
				}
				if !reading.Timestamp.IsZero() {
					t.Fatalf("Timestamp = %v, want zero time", reading.Timestamp)
				}
				assertReadingFloat32Nil(t, "WindSpeed", reading.WindSpeed)
				assertReadingFloat32Nil(t, "Pressure", reading.Pressure)
				assertReadingFloat32Nil(t, "Rain1h", reading.Rain1h)
				assertReadingInt32Nil(t, "PM1_0", reading.PM1_0)
				assertReadingInt32Nil(t, "RainAGCVariance", reading.RainAGCVariance)
			},
		},
		{
			name:    "partial json leaves omitted optional fields nil",
			payload: `{"node_id":"n1","pressure":1013.25}`,
			check: func(t *testing.T, reading Reading) {
				if reading.NodeID != "n1" {
					t.Fatalf("NodeID = %q, want n1", reading.NodeID)
				}
				if !reading.Timestamp.IsZero() {
					t.Fatalf("Timestamp = %v, want zero time", reading.Timestamp)
				}
				assertReadingFloat32Ptr(t, "Pressure", reading.Pressure, 1013.25)
				assertReadingFloat32Nil(t, "Temperature", reading.Temperature)
				assertReadingFloat32Nil(t, "Rain1h", reading.Rain1h)
				assertReadingInt32Nil(t, "PM2_5", reading.PM2_5)
			},
		},
		{
			name:    "invalid json syntax returns error",
			payload: `{bad`,
			wantErr: true,
		},
		{
			name:    "json array returns error",
			payload: `[]`,
			wantErr: true,
		},
		{
			name:    "invalid timestamp format returns error",
			payload: `{"node_id":"n1","timestamp":"not-a-time"}`,
			wantErr: true,
		},
		{
			name:    "invalid alias ts type returns error",
			payload: `{"node_id":"n1","ts":123}`,
			wantErr: true,
		},
		{
			name:    "float for integer pm1 returns error",
			payload: `{"node_id":"n1","pm1":1.5}`,
			wantErr: true,
		},
		{
			name:    "string for float temperature returns error",
			payload: `{"node_id":"n1","temperature":"hot"}`,
			wantErr: true,
		},
		{
			name:    "number for string node_id returns error",
			payload: `{"node_id":42}`,
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var reading Reading
			err := json.Unmarshal([]byte(testCase.payload), &reading)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("Unmarshal() want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal() unexpected error = %v", err)
			}
			testCase.check(t, reading)
		})
	}
}
