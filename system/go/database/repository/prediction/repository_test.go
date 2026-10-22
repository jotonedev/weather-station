package prediction

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestExtractHourlyValues(t *testing.T) {
	timestamp := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	rainfall := float32(1.5)
	tests := []struct {
		name       string
		prediction HourlyPrediction
		wantLen    int
		wantNode   string
	}{
		{
			name:       "present metrics",
			prediction: HourlyPrediction{NodeID: "n1", Ts: timestamp, HorizonHours: 6, RainfallMm: &rainfall},
			wantLen:    7,
			wantNode:   "n1",
		},
		{
			name:       "absent metrics are nil",
			prediction: HourlyPrediction{NodeID: "n2", Ts: timestamp, HorizonHours: 24},
			wantLen:    7,
			wantNode:   "n2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := extractHourlyValues(tt.prediction)
			if len(values) != tt.wantLen {
				t.Fatalf("len(values) = %d, want %d", len(values), tt.wantLen)
			}
			if values[0] != tt.wantNode {
				t.Errorf("node_id = %v, want %q", values[0], tt.wantNode)
			}
			if values[2] != tt.prediction.HorizonHours {
				t.Errorf("horizon = %v, want %d", values[2], tt.prediction.HorizonHours)
			}
		})
	}
}

func TestExtractDailyValues(t *testing.T) {
	timestamp := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	rainfall := float32(3.2)
	prediction := DailyPrediction{NodeID: "n1", Ts: timestamp, HorizonDays: 2, RainfallMm: &rainfall}
	values := extractDailyValues(prediction)

	if len(values) != 9 {
		t.Fatalf("len(values) = %d, want 9", len(values))
	}
	if values[0] != "n1" {
		t.Errorf("node_id = %v, want %q", values[0], "n1")
	}
	if values[2] != prediction.HorizonDays {
		t.Errorf("horizon = %v, want %d", values[2], prediction.HorizonDays)
	}
	if value, ok := values[7].(float32); !ok || value != rainfall {
		t.Errorf("rainfall = %v, want %v", values[7], rainfall)
	}
	for _, columnIndex := range []int{3, 4, 5, 6, 8} {
		if values[columnIndex] != nil {
			t.Errorf("index %d should be nil, got %v", columnIndex, values[columnIndex])
		}
	}
}

func TestStoreHourlyPredictionsEmpty(t *testing.T) {
	repo := &Repository{client: nil, log: zap.NewNop()}
	stored, err := repo.StoreHourlyPredictions(context.Background(), nil)
	if err != nil {
		t.Fatalf("StoreHourlyPredictions(empty): %v", err)
	}
	if stored != 0 {
		t.Errorf("stored = %d, want 0", stored)
	}
}

func TestStoreDailyPredictionsEmpty(t *testing.T) {
	repo := &Repository{client: nil, log: zap.NewNop()}
	stored, err := repo.StoreDailyPredictions(context.Background(), nil)
	if err != nil {
		t.Fatalf("StoreDailyPredictions(empty): %v", err)
	}
	if stored != 0 {
		t.Errorf("stored = %d, want 0", stored)
	}
}
