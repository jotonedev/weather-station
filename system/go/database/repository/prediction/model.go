package prediction

import "time"

// HourlyPrediction is a stored forecast point at hourly granularity.
type HourlyPrediction struct {
	NodeID       string
	Ts           time.Time
	HorizonHours int32

	TemperatureC       *float32
	HumidityPct        *float32
	RainfallMm         *float32
	RainfallConfidence *float32
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
	RainfallConfidence *float32
}

// Forecast holds the latest hourly and daily prediction batches for a node.
type Forecast struct {
	Hourly []HourlyPrediction
	Daily  []DailyPrediction
}
