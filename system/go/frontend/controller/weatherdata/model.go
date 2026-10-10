package weatherdata

import "time"

// ReadingView represents a formatted weather station reading for display.
type ReadingView struct {
	NodeID        string
	Timestamp     time.Time
	FormattedTime string

	// Climate
	Temperature string
	Humidity    string
	Pressure    string

	// Wind
	WindSpeed    string
	WindAngle    string
	RawWindSpeed string
	RawWindAngle string

	// Air Quality
	PM1_0  string
	PM2_5  string
	PM10_0 string
	VOC    string

	// Rain
	Rain1h   string
	Rain24h  string
	RainRate string

	// Diagnostics
	AnemoStatus string
	AnemoAGC    string
	AnemoAGCVar string
	VaneStatus  string
	VaneAGC     string
	VaneAGCVar  string
	RainStatus  string
	RainAGC     string
	RainAGCVar  string
}

// DashboardData represents the weather data context retrieved from the database.
type DashboardData struct {
	Title          string
	SelectedNodeID string
	AvailableNodes []string
	LatestReading  *ReadingView
	RecentReadings []ReadingView
	ErrorMessage   string
}
