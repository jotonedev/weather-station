package timestorage

import "time"

// Reading is a single raw weather sample for the weather table.
// Measurement columns are pointers so an absent field is stored as NULL
// (nodes may provide different subsets of fields).
type Reading struct {
	NodeID    string
	Timestamp time.Time

	WindSpeed    *float64
	WindAngle    *float64
	RawWindSpeed *float64
	RawWindAngle *float64
	Temperature  *float64
	Humidity     *float64
	Pressure     *float64
	Rain1h       *float64
	Rain24h      *float64
	RainRate     *float64
	PM1_0        *int32
	PM2_5        *int32
	PM10_0       *int32
	VOC          *int32
	AnemoStatus  *int32
	AnemoAGC     *int32
	AnemoAGCVar  *int32
	VaneStatus   *int32
	VaneAGC      *int32
	VaneAGCVar   *int32
	RainStatus   *int32
	RainAGC      *int32
	RainAGCVar   *int32
}

// weatherColumns lists the CopyFrom column order (node_id and ts first).
var weatherColumns = []string{
	"node_id", "ts",
	"wind_speed", "wind_angle", "raw_wind_speed", "raw_wind_angle",
	"temperature", "humidity", "pressure",
	"rain_1h", "rain_24h", "rain_rate",
	"pm1_0", "pm2_5", "pm10_0", "voc",
	"anemo_status", "anemo_agc", "anemo_agc_variance",
	"vane_status", "vane_agc", "vane_agc_variance",
	"rain_status", "rain_agc", "rain_agc_variance",
}
