package timestorage

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestExtractValues(t *testing.T) {
	timestamp := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	temperature := float32(21.5)
	pm2_5 := int32(12)

	tests := []struct {
		name       string
		reading    Reading
		wantLen    int
		wantFirst  any
		wantSecond any
	}{
		{
			name:       "full reading",
			reading:    Reading{NodeID: "n1", Timestamp: timestamp, Temperature: &temperature, PM2_5: &pm2_5},
			wantLen:    25,
			wantFirst:  "n1",
			wantSecond: timestamp,
		},
		{
			name:    "node id and timestamp present",
			reading: Reading{NodeID: "n1", Timestamp: timestamp},
			wantLen: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := extractValues(tt.reading)
			if len(values) != tt.wantLen {
				t.Fatalf("len(values) = %d, want %d", len(values), tt.wantLen)
			}
			if tt.wantFirst != nil && values[0] != tt.wantFirst {
				t.Errorf("node_id = %v, want %v", values[0], tt.wantFirst)
			}
			if tt.wantSecond != nil && values[1] != tt.wantSecond {
				t.Errorf("timestamp = %v, want %v", values[1], tt.wantSecond)
			}
		})
	}
}

func TestExtractValuesNullAndPresentColumns(t *testing.T) {
	temperature := float32(19.0)
	reading := Reading{NodeID: "n2", Timestamp: time.Now(), Temperature: &temperature}

	values := extractValues(reading)
	if values[6] == nil {
		t.Errorf("temperature column should not be nil")
	}
	if value, ok := values[6].(float32); !ok || value != temperature {
		t.Errorf("temperature = %v, want %v", values[6], temperature)
	}
	for _, columnIndex := range []int{2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15} {
		if values[columnIndex] != nil {
			t.Errorf("index %d should be nil (absent field), got %v", columnIndex, values[columnIndex])
		}
	}
}

func TestStoreReadingsEmpty(t *testing.T) {
	repo := &Repository{client: nil, log: zap.NewNop()}
	stored, err := repo.StoreReadings(context.Background(), nil)
	if err != nil {
		t.Fatalf("StoreReadings(empty): %v", err)
	}
	if stored != 0 {
		t.Errorf("stored = %d, want 0", stored)
	}
}
