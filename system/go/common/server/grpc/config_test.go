package grpc

import "testing"

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    Config
	}{
		{name: "default", address: "", want: Config{Address: defaultAddress}},
		{name: "override", address: "127.0.0.1:9000", want: Config{Address: "127.0.0.1:9000"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("GRPC_ADDRESS", test.address)

			got := NewConfig()
			if got != test.want {
				t.Errorf("NewConfig() = %+v, want %+v", got, test.want)
			}
		})
	}
}
