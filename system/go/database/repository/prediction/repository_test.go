package prediction

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestExtractValues(t *testing.T) {
	timestamp := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	rainfall := float32(1.5)
	tests := []struct {
		name       string
		prediction Prediction
		wantLen    int
		wantNode   string
	}{
		{
			name:       "present metrics",
			prediction: Prediction{NodeID: "n1", Ts: timestamp, HorizonHours: 6, RainfallMm: &rainfall},
			wantLen:    11,
			wantNode:   "n1",
		},
		{
			name:       "absent metrics are nil",
			prediction: Prediction{NodeID: "n2", Ts: timestamp, HorizonHours: 24},
			wantLen:    11,
			wantNode:   "n2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := extractValues(tt.prediction)
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

func TestExtractValuesNullColumns(t *testing.T) {
	rainfall := float32(3.2)
	prediction := Prediction{NodeID: "n1", Ts: time.Now(), HorizonHours: 3, RainfallMm: &rainfall}
	values := extractValues(prediction)

	if values[3] == nil {
		t.Errorf("rainfall column should not be nil")
	}
	if value, ok := values[3].(float32); !ok || value != rainfall {
		t.Errorf("rainfall = %v, want %v", values[3], rainfall)
	}
	for _, columnIndex := range []int{4, 5, 6, 7, 8, 9, 10} {
		if values[columnIndex] != nil {
			t.Errorf("index %d should be nil, got %v", columnIndex, values[columnIndex])
		}
	}
}

func TestStorePredictionsEmpty(t *testing.T) {
	repo := &Repository{client: nil, log: zap.NewNop()}
	stored, err := repo.StorePredictions(context.Background(), nil)
	if err != nil {
		t.Fatalf("StorePredictions(empty): %v", err)
	}
	if stored != 0 {
		t.Errorf("stored = %d, want 0", stored)
	}
}
