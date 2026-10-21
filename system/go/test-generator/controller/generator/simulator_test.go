package generator

import (
	"math"
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

func TestSimulatorStepLimits(t *testing.T) {
	tests := []struct {
		name        string
		nodeID      string
		start       time.Time
		maxPressure float64
		maxTemp     float64
		maxHumidity float64
	}{
		{
			name:        "autumn week at 10s ticks changes smoothly",
			nodeID:      "station-alpha",
			start:       time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC),
			maxPressure: 0.1,
			maxTemp:     0.3,
			maxHumidity: 2.0,
		},
		{
			name:        "summer week at 10s ticks changes smoothly",
			nodeID:      "station-beta",
			start:       time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			maxPressure: 0.1,
			maxTemp:     0.3,
			maxHumidity: 2.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sim := NewSimulator()
			step := 10 * time.Second
			prev := sim.NextReading(tc.nodeID, tc.start)
			for i := 1; i < int(7*24*time.Hour/step); i++ {
				reading := sim.NextReading(tc.nodeID, tc.start.Add(time.Duration(i)*step))
				if d := math.Abs(float64(reading.Pressure - prev.Pressure)); d > tc.maxPressure {
					t.Fatalf("pressure step %.3f hPa at %s exceeds %.3f", d, reading.Timestamp, tc.maxPressure)
				}
				if d := math.Abs(float64(reading.Temperature - prev.Temperature)); d > tc.maxTemp {
					t.Fatalf("temperature step %.3f C at %s exceeds %.3f", d, reading.Timestamp, tc.maxTemp)
				}
				if d := math.Abs(float64(reading.Humidity - prev.Humidity)); d > tc.maxHumidity {
					t.Fatalf("humidity step %.3f%% at %s exceeds %.3f", d, reading.Timestamp, tc.maxHumidity)
				}
				prev = reading
			}
		})
	}
}

func TestSimulatorIntervalIndependence(t *testing.T) {
	start := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	span := 3 * 24 * time.Hour

	run := func(step time.Duration) (meanTemp, meanHumidity, lastRain24h float64) {
		sim := NewSimulator()
		n := int(span / step)
		var sumTemp, sumHumidity float64
		for i := 0; i < n; i++ {
			reading := sim.NextReading("station-alpha", start.Add(time.Duration(i)*step))
			sumTemp += float64(reading.Temperature)
			sumHumidity += float64(reading.Humidity)
			lastRain24h = float64(reading.Rain24h)
		}
		return sumTemp / float64(n), sumHumidity / float64(n), lastRain24h
	}

	tests := []struct {
		name  string
		short time.Duration
		long  time.Duration
	}{
		{name: "10s and 60s ticks", short: 10 * time.Second, long: time.Minute},
		{name: "10s and 5m ticks", short: 10 * time.Second, long: 5 * time.Minute},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempA, humA, rainA := run(tc.short)
			tempB, humB, rainB := run(tc.long)
			if math.Abs(tempA-tempB) > 1.0 {
				t.Errorf("mean temperature differs: %.2f vs %.2f", tempA, tempB)
			}
			if math.Abs(humA-humB) > 4.0 {
				t.Errorf("mean humidity differs: %.2f vs %.2f", humA, humB)
			}
			if math.Abs(rainA-rainB) > 0.5 {
				t.Errorf("rain24h differs: %.2f vs %.2f", rainA, rainB)
			}
		})
	}
}

func TestSimulatorSharedRainDays(t *testing.T) {
	nodes := []string{"station-alpha", "station-beta"}
	first := dayIndex(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	cal := newRainCalendar()

	for day := first; day < first+60; day++ {
		dayStart := time.Unix(day*secondsDay, 0).UTC()
		rainy := len(cal.windows(day)) > 0
		prevRainy := len(cal.windows(day-1)) > 0

		for _, nodeID := range nodes {
			sim := NewSimulator()
			switch {
			case rainy:
				total := float64(sim.NextReading(nodeID, dayStart.Add(24*time.Hour)).Rain24h) +
					float64(sim.NextReading(nodeID, dayStart.Add(36*time.Hour)).Rain24h)
				if total <= 0 {
					t.Errorf("%s: rainy day %s produced no rain", nodeID, dayStart.Format(time.DateOnly))
				}
			case !prevRainy:
				for h := 1; h <= 22; h++ {
					reading := sim.NextReading(nodeID, dayStart.Add(time.Duration(h)*time.Hour))
					if reading.RainRate != 0 {
						t.Errorf("%s: dry day %s has rain rate %.2f at %dh", nodeID, dayStart.Format(time.DateOnly), reading.RainRate, h)
					}
				}
			}
		}
	}
}

func TestSimulatorRestartKeepsRainTotals(t *testing.T) {
	tests := []struct {
		name  string
		start time.Time
	}{
		{name: "autumn", start: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)},
		{name: "spring", start: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			running := NewSimulator()
			var last time.Time
			for i := 0; i < 2*24*6; i++ {
				last = tc.start.Add(time.Duration(i) * 10 * time.Minute)
				running.NextReading("station-alpha", last)
			}
			before := running.NextReading("station-alpha", last.Add(10*time.Minute))
			after := NewSimulator().NextReading("station-alpha", last.Add(10*time.Minute))
			if before.Rain24h != after.Rain24h || before.Rain1h != after.Rain1h {
				t.Fatalf("rain totals changed after restart: (%v, %v) vs (%v, %v)",
					before.Rain1h, before.Rain24h, after.Rain1h, after.Rain24h)
			}
		})
	}
}

func TestRainCalendarYearlyStats(t *testing.T) {
	tests := []struct {
		name string
		year int
	}{
		{name: "2026", year: 2026},
		{name: "2027", year: 2027},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cal := newRainCalendar()
			first := dayIndex(time.Date(tc.year, 1, 1, 0, 0, 0, 0, time.UTC))
			last := dayIndex(time.Date(tc.year+1, 1, 1, 0, 0, 0, 0, time.UTC))

			var rainyDays, autumnDays, autumnRainy, summerDays, summerRainy int
			var totalMM float64
			for day := first; day < last; day++ {
				windows := cal.windows(day)
				month := time.Unix(day*secondsDay, 0).UTC().Month()
				isAutumn := month >= time.September && month <= time.November
				isSummer := month >= time.June && month <= time.August
				if isAutumn {
					autumnDays++
				}
				if isSummer {
					summerDays++
				}
				if len(windows) == 0 {
					continue
				}
				rainyDays++
				if isAutumn {
					autumnRainy++
				}
				if isSummer {
					summerRainy++
				}
				for _, w := range windows {
					totalMM += w.peakRate * w.duration.Hours() / 2.0
				}
			}

			days := int(last - first)
			fraction := float64(rainyDays) / float64(days)
			if fraction < 0.25 || fraction > 0.50 {
				t.Errorf("rainy day fraction %.2f outside [0.25, 0.50]", fraction)
			}
			autumnFraction := float64(autumnRainy) / float64(autumnDays)
			summerFraction := float64(summerRainy) / float64(summerDays)
			if autumnFraction <= summerFraction {
				t.Errorf("autumn rainy fraction %.2f not above summer %.2f", autumnFraction, summerFraction)
			}
			if totalMM < 600 || totalMM > 1200 {
				t.Errorf("yearly rain %.0f mm outside [600, 1200]", totalMM)
			}
			t.Logf("rainy days %d/%d, autumn %.2f, summer %.2f, total %.0f mm", rainyDays, days, autumnFraction, summerFraction, totalMM)
		})
	}
}
