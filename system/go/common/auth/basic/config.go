package basic

import "os"

// Config holds credentials for HTTP Basic Authentication.
type Config struct {
	Username string
	Password string
}

// NewConfig reads HTTP_AUTH_USERNAME and HTTP_AUTH_PASSWORD from the environment.
func NewConfig() Config {
	return Config{
		Username: os.Getenv("HTTP_AUTH_USERNAME"),
		Password: os.Getenv("HTTP_AUTH_PASSWORD"),
	}
}
