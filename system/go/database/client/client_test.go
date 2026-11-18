package client

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type testLifecycle struct {
	hooks []fx.Hook
}

func (l *testLifecycle) Append(hook fx.Hook) {
	l.hooks = append(l.hooks, hook)
}

func (l *testLifecycle) start(ctx context.Context) error {
	for _, hook := range l.hooks {
		if hook.OnStart != nil {
			if err := hook.OnStart(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *testLifecycle) stop(ctx context.Context) error {
	for i := len(l.hooks) - 1; i >= 0; i-- {
		if l.hooks[i].OnStop != nil {
			if err := l.hooks[i].OnStop(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func countTable(t *testing.T, db *sql.DB, tableName string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM duckdb_tables() WHERE table_name = ?`, tableName).Scan(&count); err != nil {
		t.Fatalf("query duckdb_tables for %q: %v", tableName, err)
	}
	return count
}

func countIndex(t *testing.T, db *sql.DB, indexName string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM duckdb_indexes() WHERE index_name = ?`, indexName).Scan(&count); err != nil {
		t.Fatalf("query duckdb_indexes for %q: %v", indexName, err)
	}
	return count
}

func TestNewClient_WalAutocheckpoint(t *testing.T) {
	ctx := context.Background()

	refDB, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open reference duckdb: %v", err)
	}
	defer refDB.Close()

	var defaultSetting string
	if err := refDB.QueryRowContext(ctx, `SELECT current_setting('wal_autocheckpoint')`).Scan(&defaultSetting); err != nil {
		t.Fatalf("query default wal_autocheckpoint: %v", err)
	}
	if _, err := refDB.ExecContext(ctx, `PRAGMA wal_autocheckpoint = '1MB';`); err != nil {
		t.Fatalf("set reference wal_autocheckpoint: %v", err)
	}
	var expectedSetting string
	if err := refDB.QueryRowContext(ctx, `SELECT current_setting('wal_autocheckpoint')`).Scan(&expectedSetting); err != nil {
		t.Fatalf("query expected wal_autocheckpoint: %v", err)
	}
	if expectedSetting == defaultSetting {
		t.Fatalf("expected wal_autocheckpoint to differ from default %q", defaultSetting)
	}

	lc := &testLifecycle{}
	cfg := &Config{
		Path:           filepath.Join(t.TempDir(), "weather.duckdb"),
		MaxMemory:      "128MB",
		Threads:        2,
		ConnectTimeout: 5 * time.Second,
	}

	client, err := NewClient(lc, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := lc.start(ctx); err != nil {
		t.Fatalf("lifecycle start: %v", err)
	}
	defer func() {
		if err := lc.stop(ctx); err != nil {
			t.Fatalf("lifecycle stop: %v", err)
		}
	}()

	var gotSetting string
	if err := client.DB().QueryRowContext(ctx, `SELECT current_setting('wal_autocheckpoint')`).Scan(&gotSetting); err != nil {
		t.Fatalf("query client wal_autocheckpoint: %v", err)
	}
	if gotSetting != expectedSetting {
		t.Fatalf("wal_autocheckpoint = %q, want %q", gotSetting, expectedSetting)
	}
}

func TestInitSchema_FreshDatabase(t *testing.T) {
	ctx := context.Background()
	client, err := NewTestClient(ctx)
	if err != nil {
		t.Fatalf("NewTestClient: %v", err)
	}
	defer client.DB().Close()

	for _, table := range []string{"weather", "node_coordinates"} {
		if count := countTable(t, client.DB(), table); count != 1 {
			t.Errorf("expected table %q to exist, count = %d", table, count)
		}
	}
	for _, table := range []string{"hourly_predictions", "daily_predictions"} {
		if count := countTable(t, client.DB(), table); count != 0 {
			t.Errorf("expected legacy table %q to be absent, count = %d", table, count)
		}
	}
	for _, idx := range []string{"idx_weather_node_ts", "idx_hourly_predictions_node_ts", "idx_daily_predictions_node_ts"} {
		if count := countIndex(t, client.DB(), idx); count != 0 {
			t.Errorf("expected index %q to be absent, count = %d", idx, count)
		}
	}
}
