package aggregate

import "time"

// ClimateAggregate is one time bucket of aggregated climate readings for a node. Metrics are optional since a bucket may have no samples.
type ClimateAggregate struct {
	NodeID          string
	BucketStart     time.Time
	AvgTemperatureC *float32
	MinTemperatureC *float32
	MaxTemperatureC *float32
	AvgHumidityPct  *float32
	MinHumidityPct  *float32
	MaxHumidityPct  *float32
	AvgPressureHpa  *float32
	SampleCount     uint64
}

// AirQualityAggregate is one time bucket of aggregated air quality readings for a node. Metrics are optional since a bucket may have no samples.
type AirQualityAggregate struct {
	NodeID      string
	BucketStart time.Time
	AvgPm1      *float32
	AvgPm25     *float32
	AvgPm10     *float32
	AvgVoc      *float32
	SampleCount uint64
}
