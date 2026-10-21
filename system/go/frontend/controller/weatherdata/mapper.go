package weatherdata

import (
	"fmt"
	"math"
	"time"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

var cardinalDirections = []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}

func formatFloat(value *float32, format string, unit string) string {
	if value == nil {
		return "—"
	}
	formatted := fmt.Sprintf(format, *value)
	if unit != "" {
		return formatted + " " + unit
	}
	return formatted
}

func formatInt(value *int32, unit string) string {
	if value == nil {
		return "—"
	}
	formatted := fmt.Sprintf("%d", *value)
	if unit != "" {
		return formatted + " " + unit
	}
	return formatted
}

// degreesToCardinal converts a compass angle in degrees to an 8-point cardinal label and normalized rotation.
func degreesToCardinal(angle *float32) (string, int, bool) {
	if angle == nil {
		return "—", 0, false
	}
	normalized := math.Mod(float64(*angle), 360.0)
	if normalized < 0 {
		normalized += 360.0
	}
	index := int(math.Round(normalized/45.0)) % 8
	return cardinalDirections[index], int(math.Round(normalized)), true
}

// classifyWeatherCondition determines a weather icon key and human-readable label from metrics.
func classifyWeatherCondition(tempC, humidityPct, rainfallMm *float32) (string, string) {
	if rainfallMm != nil && *rainfallMm >= 0.2 {
		if tempC != nil && *tempC <= 0.5 {
			return "snow", "Snow"
		}
		if *rainfallMm >= 4.0 {
			return "heavy-rain", "Heavy Rain"
		}
		return "light-rain", "Light Rain"
	}

	if humidityPct != nil {
		if *humidityPct >= 82.0 {
			return "cloudy", "Overcast"
		}
		if *humidityPct >= 55.0 {
			return "partly-cloudy", "Partly Cloudy"
		}
	}

	return "sunny", "Clear Sky"
}

// classifyAirQuality returns an air quality label and level class from a PM2.5 reading.
func classifyAirQuality(pm25 *int32) (string, string) {
	if pm25 == nil {
		return "No Data", "unknown"
	}
	val := *pm25
	switch {
	case val <= 12:
		return "Good", "good"
	case val <= 35:
		return "Moderate", "moderate"
	case val <= 55:
		return "Sensitive", "sensitive"
	default:
		return "Poor", "unhealthy"
	}
}

// clampPercent bounds a float percentage to an integer in [0, 100].
func clampPercent(value *float32) int {
	if value == nil {
		return 0
	}
	rounded := int(math.Round(float64(*value)))
	if rounded < 0 {
		return 0
	}
	if rounded > 100 {
		return 100
	}
	return rounded
}

// averageConfidence calculates the mean confidence percentage across available forecast metrics.
func averageConfidence(pred *protodatabasev1.Prediction) (string, int) {
	var sum float64
	var count int
	for _, conf := range []*float32{
		pred.TemperatureConfidence,
		pred.HumidityConfidence,
		pred.RainfallConfidence,
		pred.WindDirectionConfidence,
	} {
		if conf != nil {
			sum += float64(*conf)
			count++
		}
	}
	if count == 0 {
		return "—", 0
	}
	pct := int(math.Round((sum / float64(count)) * 100.0))
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return fmt.Sprintf("%d%%", pct), pct
}

// formatHorizonLabels builds the primary time label and relative horizon sub-label for a forecast item.
func formatHorizonLabels(issueTime time.Time, horizonHours int32) (string, string) {
	subLabel := fmt.Sprintf("+%dh", horizonHours)
	if horizonHours >= 24 && horizonHours%24 == 0 {
		subLabel = fmt.Sprintf("+%dd", horizonHours/24)
	}
	if issueTime.IsZero() {
		return subLabel, "Forecast"
	}
	targetTime := issueTime.UTC().Add(time.Duration(horizonHours) * time.Hour)
	if horizonHours >= 24 && horizonHours%24 == 0 {
		return targetTime.Format("Mon 02 Jan"), subLabel
	}
	return targetTime.Format("15:04 UTC"), subLabel
}

// pbReadingToView converts a protobuf Reading into a template-friendly ReadingView.
func pbReadingToView(reading *protodatabasev1.Reading) ReadingView {
	if reading == nil {
		return ReadingView{}
	}

	var timestamp time.Time
	if reading.Timestamp != nil {
		timestamp = reading.Timestamp.AsTime()
	}

	formattedTime := "—"
	if !timestamp.IsZero() {
		formattedTime = timestamp.UTC().Format("2006-01-02 15:04:05 UTC")
	}

	rainIndicator := reading.RainRate
	if rainIndicator == nil {
		rainIndicator = reading.Rain1H
	}
	condIcon, condLabel := classifyWeatherCondition(reading.Temperature, reading.Humidity, rainIndicator)
	windCardinal, windRot, hasWind := degreesToCardinal(reading.WindAngle)
	aqLabel, aqLevel := classifyAirQuality(reading.Pm25)

	return ReadingView{
		NodeID:        reading.NodeId,
		Timestamp:     timestamp,
		FormattedTime: formattedTime,

		ConditionIcon:  condIcon,
		ConditionLabel: condLabel,

		// Climate
		Temperature:     formatFloat(reading.Temperature, "%.1f", "°C"),
		Humidity:        formatFloat(reading.Humidity, "%.1f", "%"),
		HumidityPercent: clampPercent(reading.Humidity),
		Pressure:        formatFloat(reading.Pressure, "%.1f", "hPa"),

		// Wind
		WindSpeed:       formatFloat(reading.WindSpeed, "%.1f", "m/s"),
		WindAngle:       formatFloat(reading.WindAngle, "%.0f", "°"),
		WindCardinal:    windCardinal,
		WindRotationDeg: windRot,
		HasWindAngle:    hasWind,
		RawWindSpeed:    formatFloat(reading.RawWindSpeed, "%.1f", "m/s"),
		RawWindAngle:    formatFloat(reading.RawWindAngle, "%.0f", "°"),

		// Air Quality
		PM1_0:           formatInt(reading.Pm1, "µg/m³"),
		PM2_5:           formatInt(reading.Pm25, "µg/m³"),
		PM10_0:          formatInt(reading.Pm10, "µg/m³"),
		VOC:             formatInt(reading.Voc, "ppb"),
		AirQualityLabel: aqLabel,
		AirQualityLevel: aqLevel,

		// Rain
		Rain1h:   formatFloat(reading.Rain1H, "%.1f", "mm"),
		Rain24h:  formatFloat(reading.Rain24H, "%.1f", "mm"),
		RainRate: formatFloat(reading.RainRate, "%.1f", "mm/h"),

		// Diagnostics
		AnemoStatus: formatInt(reading.AnemoStatus, ""),
		AnemoAGC:    formatInt(reading.AnemoAgc, ""),
		AnemoAGCVar: formatInt(reading.AnemoAgcVariance, ""),
		VaneStatus:  formatInt(reading.VaneStatus, ""),
		VaneAGC:     formatInt(reading.VaneAgc, ""),
		VaneAGCVar:  formatInt(reading.VaneAgcVariance, ""),
		RainStatus:  formatInt(reading.RainStatus, ""),
		RainAGC:     formatInt(reading.RainAgc, ""),
		RainAGCVar:  formatInt(reading.RainAgcVariance, ""),
	}
}

// pbPredictionToView converts a protobuf Prediction into a template-friendly ForecastItemView.
func pbPredictionToView(pred *protodatabasev1.Prediction) ForecastItemView {
	if pred == nil {
		return ForecastItemView{}
	}

	var issueTime time.Time
	if pred.IssueTime != nil {
		issueTime = pred.IssueTime.AsTime()
	}

	horizonLabel, subLabel := formatHorizonLabels(issueTime, pred.HorizonHours)
	condIcon, condLabel := classifyWeatherCondition(pred.TemperatureC, pred.HumidityPct, pred.RainfallMm)
	windCardinal, windRot, hasWind := degreesToCardinal(pred.WindDirectionDeg)
	confText, confPct := averageConfidence(pred)

	return ForecastItemView{
		HorizonHours:     pred.HorizonHours,
		HorizonLabel:     horizonLabel,
		TimeSubLabel:     subLabel,
		ConditionIcon:    condIcon,
		ConditionLabel:   condLabel,
		Temperature:      formatFloat(pred.TemperatureC, "%.1f", "°C"),
		Rainfall:         formatFloat(pred.RainfallMm, "%.1f", "mm"),
		Humidity:         formatFloat(pred.HumidityPct, "%.0f", "%"),
		WindDirection:    formatFloat(pred.WindDirectionDeg, "%.0f", "°"),
		WindCardinal:     windCardinal,
		WindRotationDeg:  windRot,
		HasWindDirection: hasWind,
		Confidence:       confText,
		ConfidencePct:    confPct,
	}
}
