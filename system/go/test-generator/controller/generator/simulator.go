package generator

import (
	"hash/fnv"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"weather-station/test-generator/gateway/ingest"
)

type nodeState struct {
	rng      *rand.Rand
	lastTime time.Time

	// Fixed per-node offsets.
	baseTemp      float64
	rainShift     time.Duration
	rainScale     float64
	windDirOffset float64
	pmScale       float64
	ratioPhase    float64

	// Values eased toward deterministic targets.
	cloud    float64
	wet      float64
	pm25Base float64
	vocBase  float64

	// Slow time-scaled noise.
	pressureDrift float64
	tempDrift     float64
	humidityDrift float64
	windNoise     float64
	windDirNoise  float64
	rainNoise     float64
	pmNoise       float64
	vocNoise      float64
}

// Simulator produces smooth, mostly deterministic telemetry per station node.
// All nodes share the same daily rain calendar and pressure waves.
type Simulator struct {
	mu     sync.Mutex
	states map[string]*nodeState
	rain   *rainCalendar
}

// NewSimulator creates a new stateful weather Simulator instance.
func NewSimulator() *Simulator {
	return &Simulator{
		states: make(map[string]*nodeState),
		rain:   newRainCalendar(),
	}
}

// NextReading advances the simulated weather state for nodeID at timestamp now and returns a reading.
func (s *Simulator) NextReading(nodeID string, now time.Time) ingest.ReadingPayload {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, exists := s.states[nodeID]
	if !exists {
		state = newNodeState(nodeID, now)
		s.states[nodeID] = state
	}

	var dt time.Duration
	if now.After(state.lastTime) {
		dt = now.Sub(state.lastTime)
	}
	state.lastTime = now
	rng := state.rng

	utc := now.UTC()
	hourUTC := float64(utc.Hour()) + float64(utc.Minute())/60.0 + float64(utc.Second())/3600.0
	dayOfYear := float64(utc.YearDay()) + hourUTC/24.0

	// 1. Rain from the shared calendar, shifted and scaled per node.
	baseRate := s.rain.rateAt(now.Add(-state.rainShift)) * state.rainScale
	state.rainNoise = ou(rng, state.rainNoise, dt, 20*time.Minute, 1.0)
	rainRate := baseRate * (1.0 + 0.1*clamp(state.rainNoise, -2.0, 2.0))
	rain1h := s.rainTotal(state, now, time.Hour)
	rain24h := s.rainTotal(state, now, 24*time.Hour)

	wetTarget := 1.0 - math.Exp(-baseRate/1.5)
	cloudTarget := 0.0
	if s.rain.nearRain(now.Add(-state.rainShift), 4*time.Hour, 2*time.Hour) {
		cloudTarget = 1.0
	}
	if !exists {
		state.wet = wetTarget
		state.cloud = cloudTarget
	}
	state.wet = ease(state.wet, wetTarget, dt, 45*time.Minute)
	state.cloud = ease(state.cloud, cloudTarget, dt, 90*time.Minute)

	// 2. Pressure from synoptic waves, a slow drift and a dip during rain (hPa).
	state.pressureDrift = ou(rng, state.pressureDrift, dt, 6*time.Hour, 0.6)
	pressure := clamp(1013.25+synopticWave(now)+state.pressureDrift-3.0*state.wet, 985.0, 1038.0)
	pressureTrend := (synopticWave(now.Add(time.Hour)) - synopticWave(now.Add(-time.Hour))) / 2.0 // hPa/h

	// 3. Temperature with seasonal and diurnal cycles. Clouds damp the diurnal swing and rain cools.
	diurnal := math.Sin(2.0 * math.Pi * (hourUTC - 9.0) / 24.0)
	seasonalTemp := 9.0 * math.Sin(2.0*math.Pi*(dayOfYear-105.0)/daysPerYear)
	state.tempDrift = ou(rng, state.tempDrift, dt, 3*time.Hour, 0.5)
	meanTemp := state.baseTemp + seasonalTemp + state.tempDrift - 2.0*state.wet
	temperature := clamp(meanTemp+5.5*(1.0-0.6*state.cloud)*diurnal, -15.0, 42.0)

	// 4. Relative humidity from temperature and dew point, rising toward saturation during rain.
	dewPoint := meanTemp - (6.0 - 3.0*state.cloud)
	rh := relativeHumidity(temperature, dewPoint)
	state.humidityDrift = ou(rng, state.humidityDrift, dt, 2*time.Hour, 2.0)
	humidity := clamp(rh+(97.0-rh)*state.wet+state.humidityDrift, 18.0, 100.0)

	// 5. Wind speed follows pressure changes, rain and afternoon mixing. Direction follows the passing systems.
	targetWind := 2.8 + clamp(4.0*math.Abs(pressureTrend), 0.0, 2.0) + 1.5*state.wet + math.Max(0.0, diurnal)
	state.windNoise = ou(rng, state.windNoise, dt, 20*time.Minute, 0.4)
	windSpeed := clamp(targetWind+state.windNoise, 0.0, 32.0)
	prevailing := 270.0 + 45.0*math.Tanh(pressureTrend/0.15) + state.windDirOffset
	dirSigma := 12.0 + 23.0*math.Exp(-windSpeed/1.5)
	state.windDirNoise = ou(rng, state.windDirNoise, dt, 30*time.Minute, dirSigma)
	windAngle := wrapAngle(prevailing + state.windDirNoise)
	rawWindSpeed := clamp(windSpeed*(1.0+0.08*rng.NormFloat64()), 0.0, 35.0)
	rawWindAngle := wrapAngle(windAngle + 3.0*rng.NormFloat64())

	// 6. Particulate matter and VOC with traffic peaks, winter increase and rain washout.
	winter := (math.Cos(2.0*math.Pi*(dayOfYear-15.0)/daysPerYear) + 1.0) / 2.0
	traffic := 1.0 + 0.35*hourBump(hourUTC, 8.0, 1.5) + 0.3*hourBump(hourUTC, 19.0, 2.0)
	pmTarget := 10.0 * state.pmScale * (1.0 + 0.25*winter) * traffic * (1.0 - 0.5*state.wet)
	vocTarget := 110.0 * traffic * (1.0 - 0.1*state.wet)
	if !exists {
		state.pm25Base = pmTarget
		state.vocBase = vocTarget
	}
	state.pm25Base = ease(state.pm25Base, pmTarget, dt, time.Hour)
	state.pmNoise = ou(rng, state.pmNoise, dt, 2*time.Hour, 1.5)
	pm25Value := clamp(state.pm25Base+state.pmNoise, 2.0, 95.0)
	ratioWave := math.Sin(2.0*math.Pi*float64(now.Unix())/(3.0*secondsDay) + state.ratioPhase)
	pm1 := int32(math.Round(clamp(pm25Value*(0.62+0.05*ratioWave), 1.0, pm25Value)))
	pm25 := int32(math.Round(pm25Value))
	pm10 := int32(math.Round(clamp(pm25Value*(1.55-0.15*ratioWave), float64(pm25), 180.0)))

	state.vocBase = ease(state.vocBase, vocTarget, dt, time.Hour)
	state.vocNoise = ou(rng, state.vocNoise, dt, 30*time.Minute, 4.0)
	voc := int32(math.Round(clamp(state.vocBase+state.vocNoise, 40.0, 350.0)))

	return ingest.ReadingPayload{
		NodeID:           nodeID,
		Timestamp:        utc,
		WindSpeed:        float32(round2(windSpeed)),
		WindAngle:        float32(roundAngle(windAngle)),
		RawWindSpeed:     float32(round2(rawWindSpeed)),
		RawWindAngle:     float32(roundAngle(rawWindAngle)),
		Temperature:      float32(round2(temperature)),
		Humidity:         float32(round2(humidity)),
		Pressure:         float32(round2(pressure)),
		Rain1h:           float32(round2(rain1h)),
		Rain24h:          float32(round2(rain24h)),
		RainRate:         float32(round2(rainRate)),
		PM1_0:            pm1,
		PM2_5:            pm25,
		PM10_0:           pm10,
		VOC:              voc,
		AnemoStatus:      32,
		AnemoAGC:         int32(126 + rng.IntN(6)),
		AnemoAGCVariance: int32(rng.IntN(3)),
		VaneStatus:       32,
		VaneAGC:          int32(124 + rng.IntN(6)),
		VaneAGCVariance:  int32(rng.IntN(3)),
		RainStatus:       32,
		RainAGC:          int32(128 + rng.IntN(5)),
		RainAGCVariance:  int32(rng.IntN(2)),
	}
}

// rainTotal integrates the node's deterministic rain rate over the span ending at now (mm).
func (s *Simulator) rainTotal(state *nodeState, now time.Time, span time.Duration) float64 {
	total := 0.0
	steps := int(span / rainStep)
	for i := range steps {
		t := now.Add(-time.Duration(i)*rainStep - rainStep/2 - state.rainShift)
		total += s.rain.rateAt(t) * state.rainScale * rainStep.Hours()
	}
	return total
}

func newNodeState(nodeID string, now time.Time) *nodeState {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(nodeID))
	seed := hasher.Sum64()
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))

	return &nodeState{
		rng:           rng,
		lastTime:      now,
		baseTemp:      15.0 + (rng.Float64()-0.5)*2.0,
		rainShift:     time.Duration((rng.Float64() - 0.5) * float64(time.Hour)),
		rainScale:     0.8 + rng.Float64()*0.4,
		windDirOffset: (rng.Float64() - 0.5) * 20.0,
		pmScale:       0.85 + rng.Float64()*0.3,
		ratioPhase:    rng.Float64() * 2.0 * math.Pi,
		pressureDrift: rng.NormFloat64() * 0.6,
		tempDrift:     rng.NormFloat64() * 0.5,
		humidityDrift: rng.NormFloat64() * 2.0,
		windNoise:     rng.NormFloat64() * 0.4,
		windDirNoise:  rng.NormFloat64() * 12.0,
		rainNoise:     rng.NormFloat64(),
		pmNoise:       rng.NormFloat64() * 1.5,
		vocNoise:      rng.NormFloat64() * 4.0,
	}
}

// ou advances an Ornstein-Uhlenbeck process with time constant tau and stationary deviation sigma.
// The exact discretization keeps the behavior independent of the sampling interval.
func ou(rng *rand.Rand, x float64, dt, tau time.Duration, sigma float64) float64 {
	if dt <= 0 {
		return x
	}
	decay := math.Exp(-dt.Seconds() / tau.Seconds())
	return x*decay + sigma*math.Sqrt(1.0-decay*decay)*rng.NormFloat64()
}

// ease moves x exponentially toward target with time constant tau.
func ease(x, target float64, dt, tau time.Duration) float64 {
	if dt <= 0 {
		return x
	}
	return target + (x-target)*math.Exp(-dt.Seconds()/tau.Seconds())
}

// relativeHumidity returns RH (%) from temperature and dew point using the Magnus formula.
func relativeHumidity(temp, dewPoint float64) float64 {
	const a, b = 17.625, 243.04
	return 100.0 * math.Exp(a*dewPoint/(b+dewPoint)-a*temp/(b+temp))
}

// hourBump is a Gaussian bump centered on hour center, wrapping around midnight.
func hourBump(hour, center, width float64) float64 {
	d := math.Abs(hour - center)
	if d > 12.0 {
		d = 24.0 - d
	}
	return math.Exp(-d * d / (2.0 * width * width))
}

func wrapAngle(deg float64) float64 {
	return math.Mod(math.Mod(deg, 360.0)+360.0, 360.0)
}

func roundAngle(deg float64) float64 {
	rounded := round2(deg)
	if rounded >= 360.0 {
		rounded -= 360.0
	}
	return rounded
}

func clamp(val, minVal, maxVal float64) float64 {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

func round2(val float64) float64 {
	return math.Round(val*100.0) / 100.0
}
