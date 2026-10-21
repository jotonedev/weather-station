package generator

import (
	"math"
	"math/rand/v2"
	"time"
)

const (
	// regionSeed is shared by all nodes so they agree on which days are rainy.
	regionSeed uint64 = 0x7e575eeda11ce5ed

	rainStep    = 5 * time.Minute
	secondsDay  = 86400
	daysPerYear = 365.25
	wettestDay  = 305.0 // early November
	driestDay   = 196.0 // mid July
)

type rainWindow struct {
	start    time.Time
	duration time.Duration
	peakRate float64 // mm/h at the middle of the window
}

// rainCalendar lazily generates and caches the deterministic rain windows of each day.
type rainCalendar struct {
	days map[int64][]rainWindow
}

func newRainCalendar() *rainCalendar {
	return &rainCalendar{days: make(map[int64][]rainWindow)}
}

func (c *rainCalendar) windows(day int64) []rainWindow {
	if w, ok := c.days[day]; ok {
		return w
	}
	if len(c.days) > 8 {
		for d := range c.days {
			if d < day-3 || d > day+3 {
				delete(c.days, d)
			}
		}
	}
	w := generateRainWindows(day)
	c.days[day] = w
	return w
}

// rateAt returns the deterministic regional rain rate (mm/h) at t.
func (c *rainCalendar) rateAt(t time.Time) float64 {
	day := dayIndex(t)
	rate := 0.0
	// Windows last at most 6h, so only the previous day can spill into the current one.
	for d := day - 1; d <= day; d++ {
		for _, w := range c.windows(d) {
			elapsed := t.Sub(w.start)
			if elapsed < 0 || elapsed >= w.duration {
				continue
			}
			s := math.Sin(math.Pi * elapsed.Seconds() / w.duration.Seconds())
			rate += w.peakRate * s * s
		}
	}
	return rate
}

// nearRain reports whether t falls inside a rain window extended by before and after.
func (c *rainCalendar) nearRain(t time.Time, before, after time.Duration) bool {
	day := dayIndex(t)
	for d := day - 1; d <= day+1; d++ {
		for _, w := range c.windows(d) {
			if !t.Before(w.start.Add(-before)) && t.Before(w.start.Add(w.duration+after)) {
				return true
			}
		}
	}
	return false
}

// generateRainWindows decides whether a day is rainy and, if so, when and how much it rains.
func generateRainWindows(day int64) []rainWindow {
	dayStart := time.Unix(day*secondsDay, 0).UTC()
	noon := dayStart.Add(12 * time.Hour)
	season := seasonFactor(noon)
	trough := clamp(-synopticWave(noon)/10.0, 0.0, 1.0)
	probability := clamp(0.30+0.12*season+0.30*trough, 0.05, 0.85)

	rng := rand.New(rand.NewPCG(regionSeed, uint64(day)))
	if rng.Float64() >= probability {
		return nil
	}

	// warm is 1 in the dry summer and 0 in the wet autumn.
	warm := (1.0 - season) / 2.0
	total := clamp((2.0+18.0*math.Pow(rng.Float64(), 2))*(0.7+0.6*trough), 1.0, 30.0)
	count := 1 + rng.IntN(3)
	if warm > 0.6 && count > 2 {
		count = 2
	}
	minHours := 2.0 - 1.5*warm
	maxHours := 6.0 - 4.0*warm

	windows := make([]rainWindow, 0, count)
	for range count {
		durHours := minHours + rng.Float64()*(maxHours-minHours)
		startHour := rng.Float64() * 24.0
		if rng.Float64() < warm {
			// Summer showers tend to form in the afternoon.
			startHour = 12.0 + rng.Float64()*6.0
		}
		amount := total / float64(count)
		windows = append(windows, rainWindow{
			start:    dayStart.Add(hoursDuration(startHour)),
			duration: hoursDuration(durHours),
			// The integral of peak*sin² over the window is peak*duration/2.
			peakRate: 2.0 * amount / durHours,
		})
	}
	return windows
}

// synopticWave returns the deterministic pressure anomaly (hPa) from passing highs and lows.
func synopticWave(t time.Time) float64 {
	days := float64(t.Unix()) / secondsDay
	return 6.0*math.Sin(2.0*math.Pi*days/4.5) + 4.0*math.Sin(2.0*math.Pi*days/7.3+1.3)
}

// seasonFactor is +1 at the wettest day of the year and -1 at the driest, varying smoothly between.
func seasonFactor(t time.Time) float64 {
	doy := float64(t.UTC().YearDay())
	sinceWettest := math.Mod(doy-wettestDay+daysPerYear, daysPerYear)
	drying := math.Mod(driestDay-wettestDay+daysPerYear, daysPerYear)
	if sinceWettest < drying {
		return math.Cos(math.Pi * sinceWettest / drying)
	}
	return math.Cos(math.Pi + math.Pi*(sinceWettest-drying)/(daysPerYear-drying))
}

func dayIndex(t time.Time) int64 {
	return int64(math.Floor(float64(t.Unix()) / secondsDay))
}

func hoursDuration(h float64) time.Duration {
	return time.Duration(h * float64(time.Hour))
}
