package aggregate

import (
	"weather-station/common/pb"
	repository "weather-station/database/repository/aggregate"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

// modelClimateToPB converts a repository ClimateAggregate into a proto aggregate.
func modelClimateToPB(aggregate repository.ClimateAggregate) *protodatabasev1.ClimateAggregate {
	return &protodatabasev1.ClimateAggregate{
		NodeId:              aggregate.NodeID,
		BucketStart:         pb.Time(aggregate.BucketStart),
		AvgTemperatureC:     aggregate.AvgTemperatureC,
		MinTemperatureC:     aggregate.MinTemperatureC,
		MaxTemperatureC:     aggregate.MaxTemperatureC,
		AvgHumidityPct:      aggregate.AvgHumidityPct,
		MinHumidityPct:      aggregate.MinHumidityPct,
		MaxHumidityPct:      aggregate.MaxHumidityPct,
		AvgPressureHpa:      aggregate.AvgPressureHpa,
		TotalRainfallMm:     aggregate.TotalRainfallMm,
		AvgWindSpeedMps:     aggregate.AvgWindSpeedMps,
		AvgWindDirectionDeg: aggregate.AvgWindDirectionDeg,
		SampleCount:         aggregate.SampleCount,
	}
}

// modelAirQualityToPB converts a repository AirQualityAggregate into a proto aggregate.
func modelAirQualityToPB(aggregate repository.AirQualityAggregate) *protodatabasev1.AirQualityAggregate {
	return &protodatabasev1.AirQualityAggregate{
		NodeId:      aggregate.NodeID,
		BucketStart: pb.Time(aggregate.BucketStart),
		AvgPm1:      aggregate.AvgPm1,
		AvgPm25:     aggregate.AvgPm25,
		AvgPm10:     aggregate.AvgPm10,
		AvgVoc:      aggregate.AvgVoc,
		SampleCount: aggregate.SampleCount,
	}
}
