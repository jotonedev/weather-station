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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := testCase.run()
			if testCase.want == nil {
				if got != nil {
					t.Errorf("Nullable = %v, want nil", got)
				}
				return
			}
			if got != testCase.want {
				t.Errorf("Nullable = %v, want %v", got, testCase.want)
			}
		})
	}
}
