package predictiondata

import (
	"testing"
	"time"

	"weather-station/common/pb"
	"weather-station/database/repository/prediction"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestModelPredictionToPB(t *testing.T) {
	issueTime := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	rainfall := float32(2.5)

	model := prediction.Prediction{
		NodeID:       "n1",
		Ts:           issueTime,
		HorizonHours: 6,
		RainfallMm:   &rainfall,
	}

	response := modelPredictionToPB(model)
	if response.NodeId != "n1" {
		t.Errorf("NodeId = %q, want n1", response.NodeId)
	}
	if !response.IssueTime.AsTime().Equal(issueTime) {
		t.Errorf("IssueTime = %v, want %v", response.IssueTime.AsTime(), issueTime)
	}
	if response.HorizonHours != 6 {
		t.Errorf("HorizonHours = %d, want 6", response.HorizonHours)
	}
	if response.RainfallMm == nil || *response.RainfallMm != rainfall {
		t.Errorf("RainfallMm = %v, want %v", response.RainfallMm, rainfall)
	}
	if response.TemperatureC != nil {
		t.Errorf("TemperatureC = %v, want nil", response.TemperatureC)
	}
}

func TestPBPredictionToModel(t *testing.T) {
	issueTime := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	rainfall := float32(2.5)

	pbPrediction := &protodatabasev1.Prediction{
		NodeId:       "n1",
		IssueTime:    pb.Time(issueTime),
		HorizonHours: 6,
		RainfallMm:   &rainfall,
	}

	model := pbPredictionToModel(pbPrediction)
	if model.NodeID != "n1" {
		t.Errorf("NodeID = %q, want n1", model.NodeID)
	}
	if !model.Ts.Equal(issueTime) {
		t.Errorf("Ts = %v, want %v", model.Ts, issueTime)
	}
	if model.HorizonHours != 6 {
		t.Errorf("HorizonHours = %d, want 6", model.HorizonHours)
	}
	if model.RainfallMm == nil || *model.RainfallMm != rainfall {
		t.Errorf("RainfallMm = %v, want %v", model.RainfallMm, rainfall)
	}
	if model.TemperatureC != nil {
		t.Errorf("TemperatureC = %v, want nil", model.TemperatureC)
	}
}
