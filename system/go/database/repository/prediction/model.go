package prediction

import "time"

type Prediction struct {
	NodeID       string
	Ts           time.Time
	HorizonHours int32

	RainfallMm         *float32
	TemperatureC       *float32
	MinTemperatureC    *float32
	MaxTemperatureC    *float32
	HumidityPct        *float32
	MinHumidityPct     *float32
	MaxHumidityPct     *float32
	RainfallConfidence *float32
}

var predictionColumns = []string{
	"node_id", "ts", "horizon_hours",
	"rainfall_mm", "temperature_c", "min_temperature_c", "max_temperature_c",
	"humidity_pct", "min_humidity_pct", "max_humidity_pct",
	"rainfall_confidence",
}
