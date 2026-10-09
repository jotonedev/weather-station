package readingdata

import (
	"testing"
	"time"

	"weather-station/common/pb"
	"weather-station/database/repository/timestorage"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestModelReadingToPB(t *testing.T) {
	timestamp := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	temperature := float32(21.5)
	pm2_5 := int32(12)

	model := timestorage.Reading{
		NodeID:      "n1",
		Timestamp:   timestamp,
		Temperature: &temperature,
		PM2_5:       &pm2_5,
	}

	response := modelReadingToPB(model)
	if response.NodeId != "n1" {
		t.Errorf("NodeId = %q, want n1", response.NodeId)
	}
	if want := pb.Time(timestamp); !response.Timestamp.AsTime().Equal(want.AsTime()) {
		t.Errorf("Timestamp = %v, want %v", response.Timestamp.AsTime(), want.AsTime())
	}
	if response.Temperature == nil || *response.Temperature != temperature {
		t.Errorf("Temperature = %v, want %v", response.Temperature, temperature)
	}
	if response.Pm25 == nil || *response.Pm25 != pm2_5 {
		t.Errorf("Pm25 = %v, want %v", response.Pm25, pm2_5)
	}
	if response.WindSpeed != nil {
		t.Errorf("WindSpeed = %v, want nil", response.WindSpeed)
	}
}

func TestPBReadingToModel(t *testing.T) {
	timestamp := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	temperature := float32(21.5)

	pbReading := &protodatabasev1.Reading{
		NodeId:      "n1",
		Timestamp:   pb.Time(timestamp),
		Temperature: &temperature,
	}

	model := pbReadingToModel(pbReading)
	if model.NodeID != "n1" {
		t.Errorf("NodeID = %q, want n1", model.NodeID)
	}
	if !model.Timestamp.Equal(timestamp) {
		t.Errorf("Timestamp = %v, want %v", model.Timestamp, timestamp)
	}
	if model.Temperature == nil || *model.Temperature != temperature {
		t.Errorf("Temperature = %v, want %v", model.Temperature, temperature)
	}
	if model.Rain1h != nil {
		t.Errorf("Rain1h = %v, want nil", model.Rain1h)
	}
}
