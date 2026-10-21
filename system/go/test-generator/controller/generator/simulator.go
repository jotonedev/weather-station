package generator

import (
	"hash/fnv"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"weather-station/test-generator/gateway/ingest"
)

type rainSample struct {
	timestamp time.Time
	amountMM  float64
}

type nodeState struct {
	rng           *rand.Rand
	lastTime      time.Time
	baseTemp      float64
	tempDrift     float64
	humidityDrift float64
	pressureDrift float64
	windSpeed     float64
	windAngle     float64
	raining       bool
	rainRate      float64
	rainHistory   []rainSample
	pm25Base      float64
	vocBase       float64
}

// Simulator maintains stateful weather physics per station node to produce realistic telemetry.
type Simulator struct {
	mu     sync.Mutex
	states map[string]*nodeState
}

// NewSimulator creates a new stateful weather Simulator instance.
func NewSimulator() *Simulator {
	return &Simulator{
		states: make(map[string]*nodeState),
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

	dtHours := 1.0 / 360.0
	if !state.lastTime.IsZero() && now.After(state.lastTime) {
		dtHours = now.Sub(state.lastTime).Hours()
		if dtHours > 1.0 {
			dtHours = 1.0
		}
	}
	state.lastTime = now

	// 1. Synoptic pressure wave and mean-reverting drift (hPa).
	dayFraction := float64(now.Unix()) / 86400.0
	synopticWave := 8.5 * math.Sin(2.0*math.Pi*dayFraction/4.5)
	state.pressureDrift = 0.96*state.pressureDrift + state.rng.NormFloat64()*0.35
	pressure := clamp(1013.25+synopticWave+state.pressureDrift, 985.0, 1038.0)

	// 2. Diurnal temperature cycle peaking around 15h UTC and seasonal cycle.
	hourUTC := float64(now.UTC().Hour()) + float64(now.UTC().Minute())/60.0
	diurnalTemp := 5.5 * math.Sin(2.0*math.Pi*(hourUTC-9.0)/24.0)
	dayOfYear := float64(now.UTC().YearDay())
	seasonalTemp := 9.0 * math.Sin(2.0*math.Pi*(dayOfYear-105.0)/365.25)
	state.tempDrift = 0.94*state.tempDrift + state.rng.NormFloat64()*0.25
	temperature := clamp(state.baseTemp+seasonalTemp+diurnalTemp+state.tempDrift, -15.0, 42.0)

	// 3. Precipitation Markov transition driven by low pressure.
	rainStartProb := 0.02
	if pressure < 1006.0 {
		rainStartProb = 0.12
	}
	if state.raining {
		if state.rng.Float64() < 0.15 {
			state.raining = false
			state.rainRate = 0
		} else {
			state.rainRate = clamp(state.rainRate*0.85+math.Abs(state.rng.NormFloat64())*2.2, 0.4, 28.0)
		}
	} else if state.rng.Float64() < rainStartProb {
		state.raining = true
		state.rainRate = clamp(0.8+math.Abs(state.rng.NormFloat64())*3.0, 0.5, 18.0)
	}

	if state.rainRate > 0 {
		state.rainHistory = append(state.rainHistory, rainSample{
			timestamp: now,
			amountMM:  state.rainRate * dtHours,
		})
	}
	cutoff24h := now.Add(-24 * time.Hour)
	cutoff1h := now.Add(-1 * time.Hour)
	var rain1h, rain24h float64
	kept := state.rainHistory[:0]
	for _, sample := range state.rainHistory {
		if !sample.timestamp.Before(cutoff24h) {
			kept = append(kept, sample)
			rain24h += sample.amountMM
			if !sample.timestamp.Before(cutoff1h) {
				rain1h += sample.amountMM
			}
		}
	}
	state.rainHistory = kept

	// 4. Relative humidity inversely tracks diurnal temperature and rises during rain.
	state.humidityDrift = 0.93*state.humidityDrift + state.rng.NormFloat64()*1.2
	rainBoost := 0.0
	if state.raining {
		rainBoost = 22.0
	}
	humidity := clamp(62.0-diurnalTemp*2.8+rainBoost+state.humidityDrift, 18.0, 100.0)

	// 5. Wind speed and circular wind direction.
	targetWind := 3.2
	if pressure < 1005.0 {
		targetWind = 6.8
	}
	state.windSpeed = clamp(0.88*state.windSpeed+0.12*targetWind+state.rng.NormFloat64()*0.45, 0.0, 32.0)
	state.windAngle = math.Mod(state.windAngle+state.rng.NormFloat64()*6.0+360.0, 360.0)
	rawWindSpeed := clamp(state.windSpeed*(0.98+state.rng.NormFloat64()*0.01), 0.0, 35.0)
	rawWindAngle := math.Mod(state.windAngle+state.rng.NormFloat64()*1.2+360.0, 360.0)

	// 6. Particulate matter and VOC with rain washout.
	washout := 1.0
	if state.raining {
		washout = 0.55
	}
	state.pm25Base = clamp(0.92*state.pm25Base+0.08*(12.0*washout)+state.rng.NormFloat64()*0.9, 2.0, 95.0)
	pm1 := int32(math.Round(clamp(state.pm25Base*0.62, 1.0, state.pm25Base)))
	pm25 := int32(math.Round(state.pm25Base))
	pm10 := int32(math.Round(clamp(state.pm25Base*1.55+math.Abs(state.rng.NormFloat64())*2.0, float64(pm25), 180.0)))

	state.vocBase = clamp(0.94*state.vocBase+0.06*115.0+state.rng.NormFloat64()*4.0, 40.0, 350.0)
	voc := int32(math.Round(state.vocBase))

	return ingest.ReadingPayload{
		NodeID:           nodeID,
		Timestamp:        now.UTC(),
		WindSpeed:        float32(round2(state.windSpeed)),
		WindAngle:        float32(round2(state.windAngle)),
		RawWindSpeed:     float32(round2(rawWindSpeed)),
		RawWindAngle:     float32(round2(rawWindAngle)),
		Temperature:      float32(round2(temperature)),
		Humidity:         float32(round2(humidity)),
		Pressure:         float32(round2(pressure)),
		Rain1h:           float32(round2(rain1h)),
		Rain24h:          float32(round2(rain24h)),
		RainRate:         float32(round2(state.rainRate)),
		PM1_0:            pm1,
		PM2_5:            pm25,
		PM10_0:           pm10,
		VOC:              voc,
		AnemoStatus:      32,
		AnemoAGC:         int32(126 + state.rng.IntN(6)),
		AnemoAGCVariance: int32(state.rng.IntN(3)),
		VaneStatus:       32,
		VaneAGC:          int32(124 + state.rng.IntN(6)),
		VaneAGCVariance:  int32(state.rng.IntN(3)),
		RainStatus:       32,
		RainAGC:          int32(128 + state.rng.IntN(5)),
		RainAGCVariance:  int32(state.rng.IntN(2)),
	}
}

func newNodeState(nodeID string, now time.Time) *nodeState {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(nodeID))
	seed := hasher.Sum64()
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))

	return &nodeState{
		rng:       rng,
		lastTime:  now,
		baseTemp:  15.0 + (rng.Float64()-0.5)*4.0,
		windSpeed: 2.5 + rng.Float64()*2.0,
		windAngle: rng.Float64() * 360.0,
		pm25Base:  11.0 + rng.Float64()*5.0,
		vocBase:   110.0 + rng.Float64()*20.0,
	}
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
