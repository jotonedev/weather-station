package questdb

import (
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string

		wantDSN            string
		wantMaxConns       int32
		wantMinConns       int32
		wantConnectTimeout time.Duration
		wantErr            bool
	}{
		{
			name: "defaults when env empty",
			env: map[string]string{
				"QUESTDB_DSN":                "",
				"QUESTDB_MAX_CONNS":          "",
				"QUESTDB_MIN_CONNS":          "",
				"QUESTDB_CONNECT_TIMEOUT_MS": "",
			},
			wantDSN:            defaultDSN,
			wantMaxConns:       defaultMaxConns,
			wantMinConns:       defaultMinConns,
			wantConnectTimeout: defaultTimeout,
		},
		{
			name: "full overrides",
			env: map[string]string{
				"QUESTDB_DSN":                "postgres://admin:quest@questdb:8812/qdb",
				"QUESTDB_MAX_CONNS":          "8",
				"QUESTDB_MIN_CONNS":          "2",
				"QUESTDB_CONNECT_TIMEOUT_MS": "1500",
			},
			wantDSN:            "postgres://admin:quest@questdb:8812/qdb",
			wantMaxConns:       8,
			wantMinConns:       2,
			wantConnectTimeout: 1500 * time.Millisecond,
		},
		{
			name: "partial override keeps other defaults",
			env: map[string]string{
				"QUESTDB_DSN":                "postgres://admin:quest@db:8812/qdb",
				"QUESTDB_MAX_CONNS":          "",
				"QUESTDB_MIN_CONNS":          "",
				"QUESTDB_CONNECT_TIMEOUT_MS": "",
			},
			wantDSN:            "postgres://admin:quest@db:8812/qdb",
			wantMaxConns:       defaultMaxConns,
			wantMinConns:       defaultMinConns,
			wantConnectTimeout: defaultTimeout,
		},
		{
			name: "invalid max conns",
			env: map[string]string{
				"QUESTDB_DSN":                "",
				"QUESTDB_MAX_CONNS":          "not-a-number",
				"QUESTDB_MIN_CONNS":          "",
				"QUESTDB_CONNECT_TIMEOUT_MS": "",
			},
			wantErr: true,
		},
		{
			name: "invalid min conns",
			env: map[string]string{
				"QUESTDB_DSN":                "",
				"QUESTDB_MAX_CONNS":          "",
				"QUESTDB_MIN_CONNS":          "many",
				"QUESTDB_CONNECT_TIMEOUT_MS": "",
			},
			wantErr: true,
		},
		{
			name: "invalid connect timeout",
			env: map[string]string{
				"QUESTDB_DSN":                "",
				"QUESTDB_MAX_CONNS":          "",
				"QUESTDB_MIN_CONNS":          "",
				"QUESTDB_CONNECT_TIMEOUT_MS": "soon",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg, err := NewConfig()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewConfig: %v", err)
			}
			if cfg.DSN != tt.wantDSN {
				t.Errorf("DSN = %q, want %q", cfg.DSN, tt.wantDSN)
			}
			if cfg.MaxConns != tt.wantMaxConns {
				t.Errorf("MaxConns = %d, want %d", cfg.MaxConns, tt.wantMaxConns)
			}
			if cfg.MinConns != tt.wantMinConns {
				t.Errorf("MinConns = %d, want %d", cfg.MinConns, tt.wantMinConns)
			}
			if cfg.ConnectTimeout != tt.wantConnectTimeout {
				t.Errorf("ConnectTimeout = %v, want %v", cfg.ConnectTimeout, tt.wantConnectTimeout)
			}
		})
	}
}
