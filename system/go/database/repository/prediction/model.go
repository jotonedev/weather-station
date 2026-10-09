package prediction

import "time"

// Prediction is a single forecast row. Only rainfall, temperature, humidity and
// wind direction are predicted; other metrics are absent by design.
type Prediction struct {
	NodeID       string
	Ts           time.Time
	HorizonHours int32

	RainfallMm       *float64
	TemperatureC     *float64
	HumidityPct      *float64
	WindDirectionDeg *float64
}

var predictionColumns = []string{
	"node_id", "ts", "horizon_hours",
	"rainfall_mm", "temperature_c", "humidity_pct", "wind_direction_deg",
}
