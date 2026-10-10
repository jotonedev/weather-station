package weatherdata

import (
	"fmt"
	"time"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
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

	return ReadingView{
		NodeID:        reading.NodeId,
		Timestamp:     timestamp,
		FormattedTime: formattedTime,

		// Climate
		Temperature: formatFloat(reading.Temperature, "%.1f", "°C"),
		Humidity:    formatFloat(reading.Humidity, "%.1f", "%"),
		Pressure:    formatFloat(reading.Pressure, "%.1f", "hPa"),

		// Wind
		WindSpeed:    formatFloat(reading.WindSpeed, "%.1f", "m/s"),
		WindAngle:    formatFloat(reading.WindAngle, "%.0f", "°"),
		RawWindSpeed: formatFloat(reading.RawWindSpeed, "%.1f", "m/s"),
		RawWindAngle: formatFloat(reading.RawWindAngle, "%.0f", "°"),

		// Air Quality
		PM1_0:  formatInt(reading.Pm1, "µg/m³"),
		PM2_5:  formatInt(reading.Pm25, "µg/m³"),
		PM10_0: formatInt(reading.Pm10, "µg/m³"),
		VOC:    formatInt(reading.Voc, "ppb"),

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
