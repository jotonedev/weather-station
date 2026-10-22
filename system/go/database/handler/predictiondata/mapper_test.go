package predictiondata

import (
	"testing"
	"time"

	"weather-station/common/pb"
	"weather-station/database/repository/prediction"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestModelHourlyToPB(t *testing.T) {
	issueTime := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	rainfall := float32(2.5)

	model := prediction.HourlyPrediction{
		NodeID:       "n1",
		Ts:           issueTime,
		HorizonHours: 6,
		RainfallMm:   &rainfall,
	}

	response := modelHourlyToPB(model)
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

func TestPbHourlyToModel(t *testing.T) {
	issueTime := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	rainfall := float32(2.5)

	pbPrediction := &protodatabasev1.HourlyPrediction{
		NodeId:       "n1",
		IssueTime:    pb.Time(issueTime),
		HorizonHours: 6,
		RainfallMm:   &rainfall,
	}

	model := pbHourlyToModel(pbPrediction)
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

func TestModelDailyToPB(t *testing.T) {
	issueTime := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	rainfall := float32(2.5)
	minTemp := float32(5.0)
	maxTemp := float32(15.0)

	model := prediction.DailyPrediction{
		NodeID:          "n1",
		Ts:              issueTime,
		HorizonDays:     2,
		MinTemperatureC: &minTemp,
		MaxTemperatureC: &maxTemp,
		RainfallMm:      &rainfall,
	}

	response := modelDailyToPB(model)
	if response.NodeId != "n1" {
		t.Errorf("NodeId = %q, want n1", response.NodeId)
	}
	if !response.IssueTime.AsTime().Equal(issueTime) {
		t.Errorf("IssueTime = %v, want %v", response.IssueTime.AsTime(), issueTime)
	}
	if response.HorizonDays != 2 {
		t.Errorf("HorizonDays = %d, want 2", response.HorizonDays)
	}
	if response.MinTemperatureC == nil || *response.MinTemperatureC != minTemp {
		t.Errorf("MinTemperatureC = %v, want %v", response.MinTemperatureC, minTemp)
	}
	if response.MaxTemperatureC == nil || *response.MaxTemperatureC != maxTemp {
		t.Errorf("MaxTemperatureC = %v, want %v", response.MaxTemperatureC, maxTemp)
	}
	if response.RainfallMm == nil || *response.RainfallMm != rainfall {
		t.Errorf("RainfallMm = %v, want %v", response.RainfallMm, rainfall)
	}
	if response.MinHumidityPct != nil {
		t.Errorf("MinHumidityPct = %v, want nil", response.MinHumidityPct)
	}
}

func TestPbDailyToModel(t *testing.T) {
	issueTime := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	rainfall := float32(2.5)

	pbPrediction := &protodatabasev1.DailyPrediction{
		NodeId:      "n1",
		IssueTime:   pb.Time(issueTime),
		HorizonDays: 3,
		RainfallMm:  &rainfall,
	}

	model := pbDailyToModel(pbPrediction)
	if model.NodeID != "n1" {
		t.Errorf("NodeID = %q, want n1", model.NodeID)
	}
	if !model.Ts.Equal(issueTime) {
		t.Errorf("Ts = %v, want %v", model.Ts, issueTime)
	}
	if model.HorizonDays != 3 {
		t.Errorf("HorizonDays = %d, want 3", model.HorizonDays)
	}
	if model.RainfallMm == nil || *model.RainfallMm != rainfall {
		t.Errorf("RainfallMm = %v, want %v", model.RainfallMm, rainfall)
	}
	if model.MinTemperatureC != nil {
		t.Errorf("MinTemperatureC = %v, want nil", model.MinTemperatureC)
	}
}
