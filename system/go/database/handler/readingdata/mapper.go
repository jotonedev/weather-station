package readingdata

import (
	"weather-station/common/pb"
	"weather-station/database/repository/timestorage"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

// pbReadingToModel converts a proto Reading into the repository Reading model.
func pbReadingToModel(reading *protodatabasev1.Reading) timestorage.Reading {
	return timestorage.Reading{
		NodeID:       reading.NodeId,
		Timestamp:    reading.Timestamp.AsTime(),
		WindSpeed:    reading.WindSpeed,
		WindAngle:    reading.WindAngle,
		RawWindSpeed: reading.RawWindSpeed,
		RawWindAngle: reading.RawWindAngle,
		Temperature:  reading.Temperature,
		Humidity:     reading.Humidity,
		Pressure:     reading.Pressure,
		Rain1h:       reading.Rain1H,
		Rain24h:      reading.Rain24H,
		RainRate:     reading.RainRate,
		PM1_0:        reading.Pm1,
		PM2_5:        reading.Pm25,
		PM10_0:       reading.Pm10,
		VOC:          reading.Voc,
		AnemoStatus:  reading.AnemoStatus,
		AnemoAGC:     reading.AnemoAgc,
		AnemoAGCVar:  reading.AnemoAgcVariance,
		VaneStatus:   reading.VaneStatus,
		VaneAGC:      reading.VaneAgc,
		VaneAGCVar:   reading.VaneAgcVariance,
		RainStatus:   reading.RainStatus,
		RainAGC:      reading.RainAgc,
		RainAGCVar:   reading.RainAgcVariance,
	}
}

// modelReadingToPB converts a repository Reading model into a proto Reading.
func modelReadingToPB(reading timestorage.Reading) *protodatabasev1.Reading {
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
		AnemoAgcVariance: reading.AnemoAGCVar,
		VaneStatus:       reading.VaneStatus,
		VaneAgc:          reading.VaneAGC,
		VaneAgcVariance:  reading.VaneAGCVar,
		RainStatus:       reading.RainStatus,
		RainAgc:          reading.RainAGC,
		RainAgcVariance:  reading.RainAGCVar,
	}
}
