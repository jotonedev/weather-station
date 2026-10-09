package timestorage

import "weather-station/common/value"

// readingSource adapts a []Reading into a pgx.CopyFromSource.
type readingSource struct {
	readings []Reading
	idx      int
}

func (s *readingSource) Next() bool {
	if s.idx >= len(s.readings) {
		return false
	}
	s.idx++
	return true
}

func (s *readingSource) Values() ([]any, error) {
	reading := s.readings[s.idx-1]
	return []any{
		reading.NodeID, reading.Timestamp,
		value.Nullable(reading.WindSpeed), value.Nullable(reading.WindAngle), value.Nullable(reading.RawWindSpeed), value.Nullable(reading.RawWindAngle),
		value.Nullable(reading.Temperature), value.Nullable(reading.Humidity), value.Nullable(reading.Pressure),
		value.Nullable(reading.Rain1h), value.Nullable(reading.Rain24h), value.Nullable(reading.RainRate),
		value.Nullable(reading.PM1_0), value.Nullable(reading.PM2_5), value.Nullable(reading.PM10_0), value.Nullable(reading.VOC),
		value.Nullable(reading.AnemoStatus), value.Nullable(reading.AnemoAGC), value.Nullable(reading.AnemoAGCVar),
		value.Nullable(reading.VaneStatus), value.Nullable(reading.VaneAGC), value.Nullable(reading.VaneAGCVar),
		value.Nullable(reading.RainStatus), value.Nullable(reading.RainAGC), value.Nullable(reading.RainAGCVar),
	}, nil
}

func (s *readingSource) Err() error { return nil }
