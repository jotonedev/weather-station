package prediction

import "time"

type Prediction struct {
	NodeID       string
	Ts           time.Time
	HorizonHours int32

	RainfallMm       *float32
	TemperatureC     *float32
	HumidityPct      *float32
	WindDirectionDeg *float32

	RainfallConfidence      *float32
	TemperatureConfidence   *float32
	HumidityConfidence      *float32
	WindDirectionConfidence *float32
}

var predictionColumns = []string{
	"node_id", "ts", "horizon_hours",
	"rainfall_mm", "temperature_c", "humidity_pct", "wind_direction_deg",
	"rainfall_confidence", "temperature_confidence", "humidity_confidence", "wind_direction_confidence",
}
