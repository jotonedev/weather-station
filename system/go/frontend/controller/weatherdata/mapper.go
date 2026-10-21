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

func formatWindSpeedKmH(speedMps *float32) string {
	if speedMps == nil {
		return "—"
	}
	kmh := *speedMps * 3.6
	return fmt.Sprintf("%.1f km/h", kmh)
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

// classifyWeatherCondition determines a Weather Icons class name and human-readable label from metrics.
func classifyWeatherCondition(tempC, humidityPct, rainfallMm *float32) (string, string) {
	if rainfallMm != nil && *rainfallMm >= 0.2 {
		if *rainfallMm >= 4.0 {
			return "wi-rain", "Heavy Rain"
		}
		return "wi-showers", "Light Rain"
	}

	return "wi-day-cloudy", "Clear Sky"
}

// classifyAirQuality returns an air quality label and level class from PM2.5 and PM10 readings
// using the European Air Quality Index (EAQI) worst-of logic.
func classifyAirQuality(pm25, pm10 *int32) (string, string) {
	if pm25 == nil && pm10 == nil {
		return "No Data", "unknown"
	}

	scorePM25 := 0
	if pm25 != nil {
		val := *pm25
		switch {
		case val <= 5:
			scorePM25 = 1
		case val <= 15:
			scorePM25 = 2
		case val <= 50:
			scorePM25 = 3
		case val <= 90:
			scorePM25 = 4
		case val <= 140:
			scorePM25 = 5
		default:
			scorePM25 = 6
		}
	}

	scorePM10 := 0
	if pm10 != nil {
		val := *pm10
		switch {
		case val <= 15:
			scorePM10 = 1
		case val <= 45:
			scorePM10 = 2
		case val <= 120:
			scorePM10 = 3
		case val <= 195:
			scorePM10 = 4
		case val <= 270:
			scorePM10 = 5
		default:
			scorePM10 = 6
		}
	}

	worst := scorePM25
	if scorePM10 > worst {
		worst = scorePM10
	}

	switch worst {
	case 1:
		return "Good", "good"
	case 2:
		return "Fair", "fair"
	case 3:
		return "Moderate", "moderate"
	case 4:
		return "Poor", "poor"
	case 5:
		return "Very Poor", "very-poor"
	case 6:
		return "Extremely Poor", "extremely-poor"
	default:
		return "No Data", "unknown"
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

// formatConfidence formats the rainfall confidence score as a percentage.
func formatConfidence(conf *float32) (string, int) {
	if conf == nil {
		return "—", 0
	}
	pct := int(math.Round(float64(*conf) * 100.0))
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
	var timestampISO string
	if !timestamp.IsZero() {
		utcTime := timestamp.UTC()
		formattedTime = utcTime.Format("2006-01-02 15:04:05 UTC")
		timestampISO = utcTime.Format(time.RFC3339)
	}

	rainIndicator := reading.RainRate
	if rainIndicator == nil {
		rainIndicator = reading.Rain1H
	}
	condIcon, condLabel := classifyWeatherCondition(reading.Temperature, reading.Humidity, rainIndicator)
	windCardinal, windRot, hasWind := degreesToCardinal(reading.WindAngle)
	aqLabel, aqLevel := classifyAirQuality(reading.Pm25, reading.Pm10)

	return ReadingView{
		NodeID:        reading.NodeId,
		Timestamp:     timestamp,
		TimestampISO:  timestampISO,
		FormattedTime: formattedTime,

		ConditionIcon:  condIcon,
		ConditionLabel: condLabel,

		// Climate
		Temperature:     formatFloat(reading.Temperature, "%.1f", "°C"),
		Humidity:        formatFloat(reading.Humidity, "%.1f", "%"),
		HumidityPercent: clampPercent(reading.Humidity),
		Pressure:        formatFloat(reading.Pressure, "%.1f", "hPa"),

		// Wind (converted to km/h from m/s)
		WindSpeed:       formatWindSpeedKmH(reading.WindSpeed),
		WindAngle:       formatFloat(reading.WindAngle, "%.0f", "°"),
		WindCardinal:    windCardinal,
		WindRotationDeg: windRot,
		HasWindAngle:    hasWind,
		RawWindSpeed:    formatWindSpeedKmH(reading.RawWindSpeed),
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
	isDaily := pred.HorizonHours >= 24 && pred.HorizonHours%24 == 0
	var targetTimeISO string
	if !issueTime.IsZero() {
		targetTime := issueTime.UTC().Add(time.Duration(pred.HorizonHours) * time.Hour)
		targetTimeISO = targetTime.Format(time.RFC3339)
	}
	condIcon, condLabel := classifyWeatherCondition(pred.TemperatureC, nil, pred.RainfallMm)
	confText, confPct := formatConfidence(pred.RainfallConfidence)

	var minTemp, maxTemp string
	if pred.MinTemperatureC != nil {
		minTemp = formatFloat(pred.MinTemperatureC, "%.1f", "°C")
	}
	if pred.MaxTemperatureC != nil {
		maxTemp = formatFloat(pred.MaxTemperatureC, "%.1f", "°")
	}

	return ForecastItemView{
		HorizonHours:   pred.HorizonHours,
		HorizonLabel:   horizonLabel,
		TimeSubLabel:   subLabel,
		TargetTimeISO:  targetTimeISO,
		IsDaily:        isDaily,
		ConditionIcon:  condIcon,
		ConditionLabel: condLabel,
		Temperature:    formatFloat(pred.TemperatureC, "%.1f", "°C"),
		MinTemperature: minTemp,
		MaxTemperature: maxTemp,
		Rainfall:       formatFloat(pred.RainfallMm, "%.1f", "mm"),
		Confidence:     confText,
		ConfidencePct:  confPct,
		HasConfidence:  pred.RainfallConfidence != nil,
	}
}
