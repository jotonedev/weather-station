package inference

import (
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
	protoinferencev1 "weather-station/gen/go/weather/inference/v1"
)

// ToTimeSeriesObservations converts database climate aggregates into time series observations for inference.
func ToTimeSeriesObservations(aggregates []*protodatabasev1.ClimateAggregate) []*protoinferencev1.TimeSeriesObservation {
	if len(aggregates) == 0 {
		return nil
	}

	observations := make([]*protoinferencev1.TimeSeriesObservation, 0, len(aggregates))
	for _, aggregate := range aggregates {
		if aggregate == nil {
			continue
		}

		obs := &protoinferencev1.TimeSeriesObservation{
			Timestamp:        aggregate.GetBucketStart(),
			TemperatureC:     aggregate.AvgTemperatureC,
			HumidityPct:      aggregate.AvgHumidityPct,
			PressureHpa:      aggregate.AvgPressureHpa,
			WindSpeedMps:     aggregate.AvgWindSpeedMps,
			WindDirectionDeg: aggregate.AvgWindDirectionDeg,
			RainfallMm:       aggregate.TotalRainfallMm,
		}
		observations = append(observations, obs)
	}

	return observations
}
