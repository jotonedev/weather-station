package generator

import (
	"testing"
	"time"
)

func TestSimulatorNextReading(t *testing.T) {
	sim := NewSimulator()
	start := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		nodeID string
		steps  int
	}{
		{
			name:   "single step produces bounded realistic values",
			nodeID: "station-alpha",
			steps:  1,
		},
		{
			name:   "hundred steps preserve physical invariants",
			nodeID: "station-beta",
			steps:  100,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			now := start
			for step := 0; step < tc.steps; step++ {
				now = now.Add(5 * time.Minute)
				reading := sim.NextReading(tc.nodeID, now)

				if reading.NodeID != tc.nodeID {
					t.Fatalf("NodeID = %q, want %q", reading.NodeID, tc.nodeID)
				}
				if reading.Temperature < -15.0 || reading.Temperature > 42.0 {
					t.Fatalf("Temperature %v out of range", reading.Temperature)
				}
				if reading.Humidity < 18.0 || reading.Humidity > 100.0 {
					t.Fatalf("Humidity %v out of range", reading.Humidity)
				}
				if reading.Pressure < 985.0 || reading.Pressure > 1038.0 {
					t.Fatalf("Pressure %v out of range", reading.Pressure)
				}
				if reading.WindSpeed < 0.0 || reading.WindAngle < 0.0 || reading.WindAngle >= 360.0 {
					t.Fatalf("Wind (%v, %v) out of range", reading.WindSpeed, reading.WindAngle)
				}
				if reading.Rain24h+0.01 < reading.Rain1h {
					t.Fatalf("Rain24h (%v) < Rain1h (%v)", reading.Rain24h, reading.Rain1h)
				}
				if reading.PM1_0 > reading.PM2_5 || reading.PM2_5 > reading.PM10_0 {
					t.Fatalf("PM invariant violated: pm1=%d pm25=%d pm10=%d", reading.PM1_0, reading.PM2_5, reading.PM10_0)
				}
			}
		})
	}
}
