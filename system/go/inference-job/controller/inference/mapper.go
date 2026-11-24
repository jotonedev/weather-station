package inference

import (
	"time"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
	protoinferencev1 "weather-station/gen/go/weather/inference/v1"
)

const (
	dailyRainThresholdMm  = 1.0
	hourlyRainThresholdMm = 0.2
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
			MinTemperatureC:  aggregate.MinTemperatureC,
			MaxTemperatureC:  aggregate.MaxTemperatureC,
			MinHumidityPct:   aggregate.MinHumidityPct,
			MaxHumidityPct:   aggregate.MaxHumidityPct,
			MinPressureHpa:   aggregate.MinPressureHpa,
			MaxPressureHpa:   aggregate.MaxPressureHpa,
			MaxWindSpeedMps:  aggregate.MaxWindSpeedMps,
			WindUMps:         aggregate.AvgWindUMps,
			WindVMps:         aggregate.AvgWindVMps,
			MaxRainRateMmh:   aggregate.MaxRainRateMmh,
		}
		observations = append(observations, obs)
	}

	return observations
}

// BlendTodayPrediction updates the horizon_days=0 prediction with observed today aggregates and remaining hourly forecasts.
func BlendTodayPrediction(
	dailyPreds []*protodatabasev1.DailyPrediction,
	issueTime time.Time,
	todayActual []*protodatabasev1.ClimateAggregate,
	hourlyPreds []*protodatabasev1.HourlyPrediction,
) {
	today := findTodayPrediction(dailyPreds)
	if today == nil || (len(todayActual) == 0 && len(hourlyPreds) == 0) {
		return
	}

	utcIssue := issueTime.UTC().Truncate(time.Hour)
	dayStart := utcIssue.Truncate(24 * time.Hour)
	threeHourCutoff := utcIssue.Truncate(3 * time.Hour)
	hoursElapsed := int32(utcIssue.Sub(dayStart) / time.Hour)
	hoursRemaining := 24 - hoursElapsed

	remHourly := filterRemainingHourly(hourlyPreds, hoursRemaining)
	blendTodayTemps(today, todayActual, remHourly)
	blendTodayHumidity(today, todayActual, remHourly)
	blendTodayRain(today, threeHourCutoff, hoursRemaining, todayActual, remHourly)
}

func findTodayPrediction(dailyPreds []*protodatabasev1.DailyPrediction) *protodatabasev1.DailyPrediction {
	for _, pred := range dailyPreds {
		if pred != nil && pred.GetHorizonDays() == 0 {
			return pred
		}
	}
	return nil
}

func filterRemainingHourly(hourlyPreds []*protodatabasev1.HourlyPrediction, hoursRemaining int32) []*protodatabasev1.HourlyPrediction {
	if hoursRemaining <= 0 {
		return nil
	}
	rem := make([]*protodatabasev1.HourlyPrediction, 0, len(hourlyPreds))
	for _, pred := range hourlyPreds {
		if pred != nil && pred.HorizonHours >= 1 && pred.HorizonHours <= hoursRemaining {
			rem = append(rem, pred)
		}
	}
	return rem
}

func float32PtrOf(val float32) *float32 {
	return &val
}

func blendTodayTemps(
	today *protodatabasev1.DailyPrediction,
	todayActual []*protodatabasev1.ClimateAggregate,
	remHourly []*protodatabasev1.HourlyPrediction,
) {
	obsMin, obsMax, hasObs := observedTempBounds(todayActual)
	remMin, remMax, remP10, remP90, hasRem := remainingTempBounds(remHourly)
	if !hasObs && !hasRem {
		return
	}

	var newMin, newMax, newP10, newP90 float32
	switch {
	case hasObs && hasRem:
		newMin = min(obsMin, remMin)
		newMax = max(obsMax, remMax)
		newP10 = min(obsMin, remP10)
		newP90 = max(obsMax, remP90)
	case hasObs:
		newMin, newMax = obsMin, obsMax
		if today.MinTemperatureC != nil {
			newMin = min(*today.MinTemperatureC, obsMin)
		}
		if today.MaxTemperatureC != nil {
			newMax = max(*today.MaxTemperatureC, obsMax)
		}
		newP10, newP90 = newMin, newMax
		if today.MinTemperatureCP10 != nil {
			newP10 = min(*today.MinTemperatureCP10, newMin)
		}
		if today.MaxTemperatureCP90 != nil {
			newP90 = max(*today.MaxTemperatureCP90, newMax)
		}
	default:
		newMin, newMax = remMin, remMax
		if today.MinTemperatureC != nil {
			newMin = min(*today.MinTemperatureC, remMin)
		}
		if today.MaxTemperatureC != nil {
			newMax = max(*today.MaxTemperatureC, remMax)
		}
		newP10 = min(newMin, remP10)
		if today.MinTemperatureCP10 != nil {
			newP10 = min(*today.MinTemperatureCP10, newP10)
		}
		newP90 = max(newMax, remP90)
		if today.MaxTemperatureCP90 != nil {
			newP90 = max(*today.MaxTemperatureCP90, newP90)
		}
	}

	if newMin > newMax {
		newMin, newMax = newMax, newMin
	}
	newP10 = min(newP10, newMin)
	newP90 = max(newP90, newMax)

	today.MinTemperatureC = float32PtrOf(newMin)
	today.MaxTemperatureC = float32PtrOf(newMax)
	today.MinTemperatureCP10 = float32PtrOf(newP10)
	today.MaxTemperatureCP90 = float32PtrOf(newP90)
}

func observedTempBounds(aggs []*protodatabasev1.ClimateAggregate) (minVal, maxVal float32, ok bool) {
	for _, agg := range aggs {
		if agg == nil {
			continue
		}
		lo := agg.MinTemperatureC
		if lo == nil {
			lo = agg.AvgTemperatureC
		}
		hi := agg.MaxTemperatureC
		if hi == nil {
			hi = agg.AvgTemperatureC
		}
		if lo != nil {
			if !ok || *lo < minVal {
				minVal = *lo
			}
			if !ok {
				maxVal = *lo
				ok = true
			}
		}
		if hi != nil {
			if !ok || *hi > maxVal {
				maxVal = *hi
			}
			if !ok {
				minVal = *hi
				ok = true
			}
		}
	}
	return minVal, maxVal, ok
}

func remainingTempBounds(preds []*protodatabasev1.HourlyPrediction) (minVal, maxVal, minP10, maxP90 float32, ok bool) {
	for _, pred := range preds {
		if pred.TemperatureC == nil {
			continue
		}
		temp := *pred.TemperatureC
		p10 := temp
		if pred.TemperatureCP10 != nil {
			p10 = *pred.TemperatureCP10
		}
		p90 := temp
		if pred.TemperatureCP90 != nil {
			p90 = *pred.TemperatureCP90
		}
		if !ok {
			minVal, maxVal, minP10, maxP90, ok = temp, temp, p10, p90, true
			continue
		}
		minVal = min(minVal, temp)
		maxVal = max(maxVal, temp)
		minP10 = min(minP10, p10)
		maxP90 = max(maxP90, p90)
	}
	return minVal, maxVal, minP10, maxP90, ok
}

func blendTodayHumidity(
	today *protodatabasev1.DailyPrediction,
	todayActual []*protodatabasev1.ClimateAggregate,
	remHourly []*protodatabasev1.HourlyPrediction,
) {
	obsMin, obsMax, hasObs := observedHumidityBounds(todayActual)
	remMin, remMax, hasRem := remainingHumidityBounds(remHourly)
	if !hasObs && !hasRem {
		return
	}

	var newMin, newMax float32
	switch {
	case hasObs && hasRem:
		newMin = min(obsMin, remMin)
		newMax = max(obsMax, remMax)
	case hasObs:
		newMin, newMax = obsMin, obsMax
		if today.MinHumidityPct != nil {
			newMin = min(*today.MinHumidityPct, obsMin)
		}
		if today.MaxHumidityPct != nil {
			newMax = max(*today.MaxHumidityPct, obsMax)
		}
	default:
		newMin, newMax = remMin, remMax
		if today.MinHumidityPct != nil {
			newMin = min(*today.MinHumidityPct, remMin)
		}
		if today.MaxHumidityPct != nil {
			newMax = max(*today.MaxHumidityPct, remMax)
		}
	}

	newMin = max(float32(0.0), min(float32(100.0), newMin))
	newMax = max(float32(0.0), min(float32(100.0), newMax))
	if newMin > newMax {
		newMin, newMax = newMax, newMin
	}
	today.MinHumidityPct = float32PtrOf(newMin)
	today.MaxHumidityPct = float32PtrOf(newMax)
}

func observedHumidityBounds(aggs []*protodatabasev1.ClimateAggregate) (minVal, maxVal float32, ok bool) {
	for _, agg := range aggs {
		if agg == nil {
			continue
		}
		lo := agg.MinHumidityPct
		if lo == nil {
			lo = agg.AvgHumidityPct
		}
		hi := agg.MaxHumidityPct
		if hi == nil {
			hi = agg.AvgHumidityPct
		}
		if lo != nil {
			if !ok || *lo < minVal {
				minVal = *lo
			}
			if !ok {
				maxVal = *lo
				ok = true
			}
		}
		if hi != nil {
			if !ok || *hi > maxVal {
				maxVal = *hi
			}
			if !ok {
				minVal = *hi
				ok = true
			}
		}
	}
	return minVal, maxVal, ok
}

func remainingHumidityBounds(preds []*protodatabasev1.HourlyPrediction) (minVal, maxVal float32, ok bool) {
	for _, pred := range preds {
		if pred.HumidityPct == nil {
			continue
		}
		hum := *pred.HumidityPct
		if !ok {
			minVal, maxVal, ok = hum, hum, true
			continue
		}
		minVal = min(minVal, hum)
		maxVal = max(maxVal, hum)
	}
	return minVal, maxVal, ok
}

func blendTodayRain(
	today *protodatabasev1.DailyPrediction,
	threeHourCutoff time.Time,
	hoursRemaining int32,
	todayActual []*protodatabasev1.ClimateAggregate,
	remHourly []*protodatabasev1.HourlyPrediction,
) {
	var obsRainMm, obsRainSince3h float64
	for _, agg := range todayActual {
		if agg == nil || agg.TotalRainfallMm == nil || *agg.TotalRainfallMm <= 0 {
			continue
		}
		rain := float64(*agg.TotalRainfallMm)
		obsRainMm += rain
		if agg.BucketStart != nil && !agg.BucketStart.AsTime().Before(threeHourCutoff) {
			obsRainSince3h += rain
		}
	}

	var remRainMm, remRainP90, maxRemProb, sumRemProb float64
	for _, pred := range remHourly {
		if pred.RainfallMm != nil && *pred.RainfallMm > 0 {
			remRainMm += float64(*pred.RainfallMm)
		}
		if pred.RainfallMmP90 != nil && *pred.RainfallMmP90 > 0 {
			remRainP90 += float64(*pred.RainfallMmP90)
		}
		if pred.RainProbability != nil {
			p := float64(*pred.RainProbability)
			maxRemProb = max(maxRemProb, p)
			sumRemProb += p
		}
	}

	totalRain := obsRainMm + remRainMm
	if len(remHourly) == 0 && today.RainfallMm != nil {
		totalRain = max(obsRainMm, float64(*today.RainfallMm))
	}
	p90 := max(totalRain, obsRainMm+remRainP90)
	today.RainfallMm = float32PtrOf(float32(totalRain))
	today.RainfallMmP90 = float32PtrOf(float32(p90))

	prob := float64(today.GetRainProbability())
	if obsRainMm >= dailyRainThresholdMm {
		prob = 1.0
	} else {
		if obsRainSince3h > 0 {
			progress := min(1.0, obsRainSince3h/dailyRainThresholdMm)
			prob = max(prob, 1.0-(1.0-prob)*(1.0-progress))
		}
		if len(remHourly) > 0 && totalRain >= dailyRainThresholdMm {
			prob = max(prob, max(0.5, maxRemProb))
		}
		needed := dailyRainThresholdMm - obsRainMm
		if len(remHourly) > 0 && hoursRemaining > 0 && needed/float64(hoursRemaining) >= hourlyRainThresholdMm {
			prob = min(prob, min(1.0, sumRemProb))
		}
	}

	prob = max(0.0, min(1.0, prob))
	today.RainProbability = float32PtrOf(float32(prob))
}
