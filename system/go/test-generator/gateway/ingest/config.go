package ingest

import "os"

// Config holds HTTP connection and authentication settings for the data-ingest endpoint.
type Config struct {
	URL      string
	Username string
	Password string
}

const defaultIngestURL = "http://ws-data-ingest:8080/readings"

// NewConfig reads INGEST_URL, HTTP_AUTH_USERNAME, and HTTP_AUTH_PASSWORD from the environment.
func NewConfig() Config {
	url := defaultIngestURL
	if envURL := os.Getenv("INGEST_URL"); envURL != "" {
		url = envURL
	}
	return Config{
		URL:      url,
		Username: os.Getenv("HTTP_AUTH_USERNAME"),
		Password: os.Getenv("HTTP_AUTH_PASSWORD"),
	}
}
