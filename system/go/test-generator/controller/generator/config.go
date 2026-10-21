package generator

import (
	"os"
	"strings"
	"time"
)

// Config holds settings for synthetic weather reading generation.
type Config struct {
	Interval time.Duration
	NodeIDs  []string
}

const (
	defaultInterval = 10 * time.Second
	defaultNodeID   = "station-alpha"
)

// NewConfig reads GENERATOR_INTERVAL and GENERATOR_NODE_IDS from the environment.
func NewConfig() (Config, error) {
	interval := defaultInterval
	if rawInterval := strings.TrimSpace(os.Getenv("GENERATOR_INTERVAL")); rawInterval != "" {
		parsed, err := time.ParseDuration(rawInterval)
		if err != nil {
			return Config{}, err
		}
		if parsed > 0 {
			interval = parsed
		}
	}

	var nodeIDs []string
	if rawNodes := strings.TrimSpace(os.Getenv("GENERATOR_NODE_IDS")); rawNodes != "" {
		for _, part := range strings.Split(rawNodes, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				nodeIDs = append(nodeIDs, trimmed)
			}
		}
	}
	if len(nodeIDs) == 0 {
		nodeIDs = []string{defaultNodeID}
	}

	return Config{
		Interval: interval,
		NodeIDs:  nodeIDs,
	}, nil
}
