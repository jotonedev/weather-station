package timestorage

import "time"

type Reading struct {
	NodeID    string
	Timestamp time.Time

	WindSpeed    *float32
	WindAngle    *float32
	RawWindSpeed *float32
	RawWindAngle *float32
	Temperature  *float32
	Humidity     *float32
	Pressure     *float32
	Rain1h       *float32
	Rain24h      *float32
	RainRate     *float32
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
