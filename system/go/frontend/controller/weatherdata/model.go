package weatherdata

import "time"

// ReadingView represents a formatted weather station reading for display.
type ReadingView struct {
	NodeID        string
	Timestamp     time.Time
	FormattedTime string

	// Derived condition
	ConditionIcon  string
	ConditionLabel string

	// Climate
	Temperature     string
	Humidity        string
	HumidityPercent int
	Pressure        string

	// Wind
	WindSpeed       string
	WindAngle       string
	WindCardinal    string
	WindRotationDeg int
	HasWindAngle    bool
	RawWindSpeed    string
	RawWindAngle    string

	// Air Quality
	PM1_0           string
	PM2_5           string
	PM10_0          string
	VOC             string
	AirQualityLabel string
	AirQualityLevel string

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

// ForecastItemView represents a single horizon forecast prediction for display.
type ForecastItemView struct {
	HorizonHours     int32
	HorizonLabel     string
	TimeSubLabel     string
	ConditionIcon    string
	ConditionLabel   string
	Temperature      string
	Rainfall         string
	Humidity         string
	WindDirection    string
	WindCardinal     string
	WindRotationDeg  int
	HasWindDirection bool
	Confidence       string
	ConfidencePct    int
}

// DashboardData represents the weather data context retrieved from the database.
type DashboardData struct {
	Title            string
	SelectedNodeID   string
	AvailableNodes   []string
	NoNodes          bool
	LatestReading    *ReadingView
	Forecast         []ForecastItemView
	ForecastIssuedAt string
	ErrorMessage     string
}
