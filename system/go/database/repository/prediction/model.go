package prediction

import "time"

// HourlyPrediction is a stored forecast point at hourly granularity.
type HourlyPrediction struct {
	NodeID       string
	Ts           time.Time
	HorizonHours int32

	TemperatureC    *float32
	HumidityPct     *float32
	RainfallMm      *float32
	RainProbability *float32
	TemperatureCP10 *float32
	TemperatureCP90 *float32
	HumidityPctP10  *float32
	HumidityPctP90  *float32
	RainfallMmP90   *float32
	PressureHpa     *float32
	PressureHpaP10  *float32
	PressureHpaP90  *float32
}

// DailyPrediction is a stored forecast point at daily granularity.
type DailyPrediction struct {
	NodeID      string
	Ts          time.Time
	HorizonDays int32

	MinTemperatureC    *float32
	MaxTemperatureC    *float32
	MinHumidityPct     *float32
	MaxHumidityPct     *float32
	RainfallMm         *float32
	RainProbability    *float32
	MinTemperatureCP10 *float32
	MaxTemperatureCP90 *float32
	RainfallMmP90      *float32
	PressureHpa        *float32
	PressureHpaP10     *float32
	PressureHpaP90     *float32
}

// Forecast holds the latest hourly and daily prediction batches for a node.
type Forecast struct {
	Hourly []HourlyPrediction
	Daily  []DailyPrediction
}
