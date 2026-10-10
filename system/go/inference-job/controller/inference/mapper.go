package inference

import (
	"encoding/binary"
	"math"

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
			Timestamp:      aggregate.GetBucketStart(),
			TemperatureC:   aggregate.AvgTemperatureC,
			HumidityPct:    aggregate.AvgHumidityPct,
			PressureHpa:    aggregate.AvgPressureHpa,
		}
		observations = append(observations, obs)
	}

	return observations
}

// PackFloat32TensorBlob packs temperature, humidity, and pressure into a contiguous float32 little-endian byte buffer.
// Missing metrics are packed as NaN (IEEE 754 float32).
func PackFloat32TensorBlob(observations []*protoinferencev1.TimeSeriesObservation) []byte {
	if len(observations) == 0 {
		return nil
	}

	const channels = 3
	buffer := make([]byte, len(observations)*channels*4)
	offset := 0

	for _, obs := range observations {
		temp := float32(math.NaN())
		if obs.TemperatureC != nil {
			temp = *obs.TemperatureC
		}

		humidity := float32(math.NaN())
		if obs.HumidityPct != nil {
			humidity = *obs.HumidityPct
		}

		pressure := float32(math.NaN())
		if obs.PressureHpa != nil {
			pressure = *obs.PressureHpa
		}

		binary.LittleEndian.PutUint32(buffer[offset:], math.Float32bits(temp))
		offset += 4
		binary.LittleEndian.PutUint32(buffer[offset:], math.Float32bits(humidity))
		offset += 4
		binary.LittleEndian.PutUint32(buffer[offset:], math.Float32bits(pressure))
		offset += 4
	}

	return buffer
}
