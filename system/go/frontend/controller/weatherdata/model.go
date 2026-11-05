package weatherdata

// ReadingView represents a formatted weather station reading for display.
type ReadingView struct {
	TimestampISO  string
	FormattedTime string

	// Derived condition
	ConditionIcon  string
	ConditionLabel string

	// Climate
	Temperature string
	Humidity    string
	Pressure    string

	// Wind
	WindSpeed       string
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
	HorizonLabel       string
	TimeSubLabel       string
	TargetTimeISO      string
	ConditionIcon      string
	ConditionLabel     string
	Temperature        string
	TemperatureRange   string
	MinTemperature     string
	MaxTemperature     string
	Rainfall           string
	RainProbability    string
	RainProbabilityPct int
}

// DashboardData represents the weather data context retrieved from the database.
type DashboardData struct {
	Title             string
	SelectedNodeID    string
	AvailableNodes    []string
	NoNodes           bool
	LatestReading     *ReadingView
	HourlyForecast    []ForecastItemView
	DailyForecast     []ForecastItemView
	HourlyIssuedAt    string
	HourlyIssuedAtISO string
	DailyIssuedAt     string
	DailyIssuedAtISO  string
	ErrorMessage      string
}
