package timestorage

import (
	"weather-station/common/value"
)

// rowScanner abstracts *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func extractValues(reading Reading) []any {
	return []any{
		reading.NodeID, reading.Timestamp,
		value.Nullable(reading.WindSpeed), value.Nullable(reading.WindAngle), value.Nullable(reading.RawWindSpeed), value.Nullable(reading.RawWindAngle),
		value.Nullable(reading.Temperature), value.Nullable(reading.Humidity), value.Nullable(reading.Pressure),
		value.Nullable(reading.Rain1h), value.Nullable(reading.Rain24h), value.Nullable(reading.RainRate),
		value.Nullable(reading.PM1_0), value.Nullable(reading.PM2_5), value.Nullable(reading.PM10_0), value.Nullable(reading.VOC),
		value.Nullable(reading.AnemoStatus), value.Nullable(reading.AnemoAGC), value.Nullable(reading.AnemoAGCVar),
		value.Nullable(reading.VaneStatus), value.Nullable(reading.VaneAGC), value.Nullable(reading.VaneAGCVar),
		value.Nullable(reading.RainStatus), value.Nullable(reading.RainAGC), value.Nullable(reading.RainAGCVar),
	}
}

// scanReading reads a weather row into a Reading, using a nil field for a NULL source column.
func scanReading(row rowScanner) (Reading, error) {
	var reading Reading
	err := row.Scan(
		&reading.NodeID, &reading.Timestamp,
		&reading.WindSpeed, &reading.WindAngle, &reading.RawWindSpeed, &reading.RawWindAngle,
		&reading.Temperature, &reading.Humidity, &reading.Pressure,
		&reading.Rain1h, &reading.Rain24h, &reading.RainRate,
		&reading.PM1_0, &reading.PM2_5, &reading.PM10_0, &reading.VOC,
		&reading.AnemoStatus, &reading.AnemoAGC, &reading.AnemoAGCVar,
		&reading.VaneStatus, &reading.VaneAGC, &reading.VaneAGCVar,
		&reading.RainStatus, &reading.RainAGC, &reading.RainAGCVar,
	)
	return reading, err
}
