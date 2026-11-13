package weatherdata

import (
	"fmt"
	"math"
	"time"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

var (
	cardinalDirections = []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	pm25Thresholds     = [5]int32{5, 15, 50, 90, 140}
	pm10Thresholds     = [5]int32{15, 45, 120, 195, 270}
	aqLabels           = [7][2]string{
		{"No Data", "unknown"},
		{"Good", "good"},
		{"Fair", "fair"},
		{"Moderate", "moderate"},
		{"Poor", "poor"},
		{"Very Poor", "very-poor"},
		{"Extremely Poor", "extremely-poor"},
	}
)

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
	return fmt.Sprintf("%.1f km/h", *speedMps*3.6)
}

func formatRange(low, high *float32, format string, unit string) string {
	if low == nil || high == nil {
		return ""
	}
	formatted := fmt.Sprintf(format+"–"+format, *low, *high)
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

// classifyWeatherCondition determines a Weather Icons class name and human-readable label from rainfall.
func classifyWeatherCondition(rainfallMm *float32) (string, string) {
	if rainfallMm != nil && *rainfallMm >= 0.2 {
		if *rainfallMm >= 4.0 {
			return "wi-rain", "Heavy Rain"
		}
		return "wi-showers", "Light Rain"
	}
	return "wi-day-cloudy", "Clear Sky"
}

// classifyForecastCondition determines a Weather Icons class name and label for a forecast item.
func classifyForecastCondition(rainfallMm, rainProb *float32) (string, string) {
	isRainy := (rainProb != nil && *rainProb >= 0.4) || (rainfallMm != nil && *rainfallMm >= 0.2)
	if !isRainy {
		return "wi-day-cloudy", "Clear Sky"
	}
	if rainfallMm != nil && *rainfallMm >= 4.0 {
		return "wi-rain", "Heavy Rain"
	}
	return "wi-showers", "Light Rain"
}

func scorePollutant(val *int32, thresholds [5]int32) int {
	if val == nil {
		return 0
	}
	for idx, bound := range thresholds {
		if *val <= bound {
			return idx + 1
		}
	}
	return 6
}

// classifyAirQuality returns an air quality label and level class from PM2.5 and PM10 readings
// using the European Air Quality Index (EAQI) worst-of logic.
func classifyAirQuality(pm25, pm10 *int32) (string, string) {
	worst := max(scorePollutant(pm25, pm25Thresholds), scorePollutant(pm10, pm10Thresholds))
	return aqLabels[worst][0], aqLabels[worst][1]
}

// formatProbability formats a [0, 1] probability as a percentage string and integer.
func formatProbability(prob *float32) (string, int) {
	if prob == nil {
		return "—", 0
	}
	pct := max(0, min(100, int(math.Round(float64(*prob)*100.0))))
	return fmt.Sprintf("%d%%", pct), pct
}

func formatHourlyHorizonLabel(issueTime time.Time, horizonHours int32) (string, string, string) {
	subLabel := fmt.Sprintf("+%dh", horizonHours)
	if issueTime.IsZero() {
		return subLabel, "Forecast", ""
	}
	targetTime := issueTime.UTC().Add(time.Duration(horizonHours) * time.Hour)
	return targetTime.Format("15:04 UTC"), subLabel, targetTime.Format(time.RFC3339)
}

func formatDailyHorizonLabel(issueTime time.Time, horizonDays int32) (string, string, string) {
	subLabel := fmt.Sprintf("+%dd", horizonDays)
	if horizonDays == 0 {
		subLabel = "Today"
	}
	if issueTime.IsZero() {
		return subLabel, "Forecast", ""
	}
	targetTime := issueTime.UTC().Add(time.Duration(horizonDays) * 24 * time.Hour)
	return targetTime.Format("Mon 02 Jan"), subLabel, targetTime.Format(time.RFC3339)
}

// pbReadingToView converts a protobuf Reading into a template-friendly ReadingView.
func pbReadingToView(reading *protodatabasev1.Reading) ReadingView {
	if reading == nil {
		return ReadingView{}
	}

	formattedTime := "—"
	var timestampISO string
	if reading.Timestamp != nil && !reading.Timestamp.AsTime().IsZero() {
		utcTime := reading.Timestamp.AsTime().UTC()
		formattedTime = utcTime.Format("2006-01-02 15:04:05 UTC")
		timestampISO = utcTime.Format(time.RFC3339)
	}

	rainIndicator := reading.RainRate
	if rainIndicator == nil {
		rainIndicator = reading.Rain1H
	}
	condIcon, condLabel := classifyWeatherCondition(rainIndicator)
	windCardinal, windRot, hasWind := degreesToCardinal(reading.WindAngle)
	aqLabel, aqLevel := classifyAirQuality(reading.Pm25, reading.Pm10)

	return ReadingView{
		TimestampISO:   timestampISO,
		FormattedTime:  formattedTime,
		ConditionIcon:  condIcon,
		ConditionLabel: condLabel,

		Temperature: formatFloat(reading.Temperature, "%.1f", "°C"),
		Humidity:    formatFloat(reading.Humidity, "%.1f", "%"),
		Pressure:    formatFloat(reading.Pressure, "%.1f", "hPa"),

		WindSpeed:       formatWindSpeedKmH(reading.WindSpeed),
		WindCardinal:    windCardinal,
		WindRotationDeg: windRot,
		HasWindAngle:    hasWind,
		RawWindSpeed:    formatWindSpeedKmH(reading.RawWindSpeed),
		RawWindAngle:    formatFloat(reading.RawWindAngle, "%.0f", "°"),

		PM1_0:           formatInt(reading.Pm1, "µg/m³"),
		PM2_5:           formatInt(reading.Pm25, "µg/m³"),
		PM10_0:          formatInt(reading.Pm10, "µg/m³"),
		AirQualityLabel: aqLabel,
		AirQualityLevel: aqLevel,

		Rain1h:   formatFloat(reading.Rain1H, "%.1f", "mm"),
		Rain24h:  formatFloat(reading.Rain24H, "%.1f", "mm"),
		RainRate: formatFloat(reading.RainRate, "%.1f", "mm/h"),

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

// pbHourlyToView converts a protobuf HourlyPrediction into a template-friendly ForecastItemView.
func pbHourlyToView(pred *protodatabasev1.HourlyPrediction) ForecastItemView {
	if pred == nil {
		return ForecastItemView{}
	}

	var issueTime time.Time
	if pred.IssueTime != nil {
		issueTime = pred.IssueTime.AsTime()
	}

	horizonLabel, subLabel, targetTimeISO := formatHourlyHorizonLabel(issueTime, pred.HorizonHours)
	condIcon, condLabel := classifyForecastCondition(pred.RainfallMm, pred.RainProbability)
	probText, probPct := formatProbability(pred.RainProbability)

	return ForecastItemView{
		HorizonLabel:       horizonLabel,
		TimeSubLabel:       subLabel,
		TargetTimeISO:      targetTimeISO,
		ConditionIcon:      condIcon,
		ConditionLabel:     condLabel,
		Temperature:        formatFloat(pred.TemperatureC, "%.1f", "°C"),
		TemperatureRange:   formatRange(pred.TemperatureCP10, pred.TemperatureCP90, "%.1f", "°C"),
		Rainfall:           formatFloat(pred.RainfallMm, "%.1f", "mm"),
		RainProbability:    probText,
		RainProbabilityPct: probPct,
	}
}

// pbDailyToView converts a protobuf DailyPrediction into a template-friendly ForecastItemView.
func pbDailyToView(pred *protodatabasev1.DailyPrediction) ForecastItemView {
	if pred == nil {
		return ForecastItemView{}
	}

	var issueTime time.Time
	if pred.IssueTime != nil {
		issueTime = pred.IssueTime.AsTime()
	}

	horizonLabel, subLabel, targetTimeISO := formatDailyHorizonLabel(issueTime, pred.HorizonDays)
	condIcon, condLabel := classifyForecastCondition(pred.RainfallMm, pred.RainProbability)
	probText, probPct := formatProbability(pred.RainProbability)

	var minTemp, maxTemp string
	if pred.MinTemperatureC != nil {
		minTemp = formatFloat(pred.MinTemperatureC, "%.1f", "°C")
	}
	if pred.MaxTemperatureC != nil {
		maxTemp = formatFloat(pred.MaxTemperatureC, "%.1f", "°C")
	}

	return ForecastItemView{
		HorizonLabel:       horizonLabel,
		TimeSubLabel:       subLabel,
		TargetTimeISO:      targetTimeISO,
		ConditionIcon:      condIcon,
		ConditionLabel:     condLabel,
		MinTemperature:     minTemp,
		MaxTemperature:     maxTemp,
		Rainfall:           formatFloat(pred.RainfallMm, "%.1f", "mm"),
		RainProbability:    probText,
		RainProbabilityPct: probPct,
	}
}
