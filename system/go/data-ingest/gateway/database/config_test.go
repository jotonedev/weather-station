package database

import "testing"

func TestNewConfig(t *testing.T) {
	testCases := []struct {
		name         string
		databaseAddr string
		grpcAddr     string
		want         Config
	}{
		{
			name: "default when no environment variable is set",
			want: Config{Address: "ws-database:8090"},
		},
		{
			name:         "service name only appends default port",
			databaseAddr: "ws-database",
			want:         Config{Address: "ws-database:8090"},
		},
		{
			name:         "custom host and port via DATABASE_ADDRESS",
			databaseAddr: "database-service:9000",
			want:         Config{Address: "database-service:9000"},
		},
		{
			name:     "custom host and port via DATABASE_GRPC_ADDRESS",
			grpcAddr: "custom-db:9090",
			want:     Config{Address: "custom-db:9090"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("DATABASE_ADDRESS", testCase.databaseAddr)
			t.Setenv("DATABASE_GRPC_ADDRESS", testCase.grpcAddr)

			got := NewConfig()
			if got != testCase.want {
				t.Fatalf("NewConfig() = %+v, want %+v", got, testCase.want)
			}
		})
	}
}
