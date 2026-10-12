package gateway

import (
	"context"
	"fmt"

	"weather-station/common/client/questdb"
)

// QuestDBChecker checks the readiness of the QuestDB connection.
type QuestDBChecker struct {
	client *questdb.Client
}

// NewQuestDBChecker creates a new QuestDBChecker.
func NewQuestDBChecker(client *questdb.Client) *QuestDBChecker {
	return &QuestDBChecker{client: client}
}

// Name returns the name of the readiness check.
func (*QuestDBChecker) Name() string {
	return "questdb"
}

// Check pings the QuestDB database pool.
func (c *QuestDBChecker) Check(ctx context.Context) error {
	if c.client == nil {
		return nil
	}
	if err := c.client.Ping(ctx); err != nil {
		return fmt.Errorf("ping questdb: %w", err)
	}
	return nil
}
