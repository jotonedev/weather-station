package weatherstation

import (
	"weather-station/common/pb"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

// readingToPB converts a controller Reading into a protobuf Reading message.
func readingToPB(reading Reading) *protodatabasev1.Reading {
	return &protodatabasev1.Reading{
		NodeId:           reading.NodeID,
		Timestamp:        pb.Time(reading.Timestamp),
		WindSpeed:        reading.WindSpeed,
		WindAngle:        reading.WindAngle,
		RawWindSpeed:     reading.RawWindSpeed,
		RawWindAngle:     reading.RawWindAngle,
		Temperature:      reading.Temperature,
		Humidity:         reading.Humidity,
		Pressure:         reading.Pressure,
		Rain1H:           reading.Rain1h,
		Rain24H:          reading.Rain24h,
		RainRate:         reading.RainRate,
		Pm1:              reading.PM1_0,
		Pm25:             reading.PM2_5,
		Pm10:             reading.PM10_0,
		Voc:              reading.VOC,
		AnemoStatus:      reading.AnemoStatus,
		AnemoAgc:         reading.AnemoAGC,
		AnemoAgcVariance: reading.AnemoAGCVariance,
		VaneStatus:       reading.VaneStatus,
		VaneAgc:          reading.VaneAGC,
		VaneAgcVariance:  reading.VaneAGCVariance,
		RainStatus:       reading.RainStatus,
		RainAgc:          reading.RainAGC,
		RainAgcVariance:  reading.RainAGCVariance,
	}
}
