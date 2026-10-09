package value

import "testing"

func TestNullable(t *testing.T) {
	temperature := 21.5
	pm2_5 := int32(12)
	var nilTemperature *float64

	tests := []struct {
		name string
		run  func() any
		want any
	}{
		{
			name: "float value",
			run:  func() any { return Nullable(&temperature) },
			want: 21.5,
		},
		{
			name: "int value",
			run:  func() any { return Nullable(&pm2_5) },
			want: int32(12),
		},
		{
			name: "nil float pointer",
			run:  func() any { return Nullable(nilTemperature) },
			want: nil,
		},
		{
			name: "nil pointer argument",
			run:  func() any { return Nullable[*float64](nil) },
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.run()
			if tt.want == nil {
				if output != nil {
					t.Errorf("Nullable = %v, want nil", output)
				}
				return
			}
			if output != tt.want {
				t.Errorf("Nullable = %v, want %v", output, tt.want)
			}
		})
	}
}
