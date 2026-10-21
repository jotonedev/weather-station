package inference

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
	protoinferencev1 "weather-station/gen/go/weather/inference/v1"
	"weather-station/inference-job/internal/mocks"
)

func TestNewController(t *testing.T) {
	logger := zap.NewNop()
	ctrl := gomock.NewController(t)
	mockDB := mocks.NewMockDatabaseGateway(ctrl)
	mockInf := mocks.NewMockInferenceGateway(ctrl)

	tests := []struct {
		name       string
		dbGateway  DatabaseGateway
		infGateway InferenceGateway
		wantErr    bool
	}{
		{
			name:       "missing gateways returns error",
			dbGateway:  nil,
			infGateway: nil,
			wantErr:    true,
		},
		{
			name:       "missing inference gateway returns error",
			dbGateway:  mockDB,
			infGateway: nil,
			wantErr:    true,
		},
		{
			name:       "valid gateways succeeds",
			dbGateway:  mockDB,
			infGateway: mockInf,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(ControllerParams{
				Logger:     logger,
				DBGateway:  tt.dbGateway,
				InfGateway: tt.infGateway,
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("New() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == nil {
				t.Fatal("expected non-nil Controller")
			}
		})
	}
}

func TestRunHourlyForecastForNode(t *testing.T) {
	logger := zap.NewNop()
	fixedTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	errDB := errors.New("db failed")
	errInf := errors.New("inference failed")
	errStore := errors.New("store failed")

	tests := []struct {
		name         string
		nodeID       string
		horizonHours int32
		setup        func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway)
		wantErr      bool
	}{
		{
			name:         "zero horizon returns ErrInvalidHorizon",
			nodeID:       "node-1",
			horizonHours: 0,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
			},
			wantErr: true,
		},
		{
			name:         "negative horizon returns ErrInvalidHorizon",
			nodeID:       "node-1",
			horizonHours: -1,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
			},
			wantErr: true,
		},
		{
			name:         "db aggregates query error returns error",
			nodeID:       "node-1",
			horizonHours: 24,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-40*24*time.Hour), fixedTime, uint32(3600)).
					Return(nil, errDB)
			},
			wantErr: true,
		},
		{
			name:         "no aggregates skips prediction and returns nil",
			nodeID:       "node-1",
			horizonHours: 24,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-40*24*time.Hour), fixedTime, uint32(3600)).
					Return([]*protodatabasev1.ClimateAggregate{}, nil)
			},
			wantErr: false,
		},
		{
			name:         "inference rpc error returns error",
			nodeID:       "node-1",
			horizonHours: 24,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-40*24*time.Hour), fixedTime, uint32(3600)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedTime.Add(-time.Hour))},
					}, nil)
				mockInf.EXPECT().
					PredictHourlyWeather(gomock.Any(), gomock.Any()).
					Return(nil, errInf)
			},
			wantErr: true,
		},
		{
			name:         "store predictions error returns error",
			nodeID:       "node-1",
			horizonHours: 24,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-40*24*time.Hour), fixedTime, uint32(3600)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedTime.Add(-time.Hour))},
					}, nil)
				mockInf.EXPECT().
					PredictHourlyWeather(gomock.Any(), gomock.Any()).
					Return(&protoinferencev1.PredictHourlyWeatherResponse{
						Predictions: []*protodatabasev1.HourlyPrediction{
							{NodeId: "node-1", HorizonHours: 1},
						},
					}, nil)
				mockDB.EXPECT().
					StorePredictions(gomock.Any(), gomock.Len(1)).
					Return(errStore)
			},
			wantErr: true,
		},
		{
			name:         "successful hourly forecast",
			nodeID:       "node-1",
			horizonHours: 24,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-40*24*time.Hour), fixedTime, uint32(3600)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedTime.Add(-time.Hour))},
					}, nil)
				mockInf.EXPECT().
					PredictHourlyWeather(gomock.Any(), gomock.AssignableToTypeOf(&protoinferencev1.PredictHourlyWeatherRequest{})).
					Return(&protoinferencev1.PredictHourlyWeatherResponse{
						Predictions: []*protodatabasev1.HourlyPrediction{
							{NodeId: "node-1", HorizonHours: 1},
						},
					}, nil)
				mockDB.EXPECT().
					StorePredictions(gomock.Any(), gomock.Len(1)).
					Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockDB := mocks.NewMockDatabaseGateway(ctrl)
			mockInf := mocks.NewMockInferenceGateway(ctrl)
			tt.setup(mockDB, mockInf)

			c := &Controller{
				log:        logger,
				dbGateway:  mockDB,
				infGateway: mockInf,
				nowFunc:    func() time.Time { return fixedTime },
			}

			err := c.RunHourlyForecastForNode(context.Background(), tt.nodeID, tt.horizonHours)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RunHourlyForecastForNode() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunDailyForecastForNode(t *testing.T) {
	logger := zap.NewNop()
	fixedTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	errDB := errors.New("db failed")
	errInf := errors.New("inference failed")
	errStore := errors.New("store failed")

	tests := []struct {
		name        string
		nodeID      string
		horizonDays int32
		setup       func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway)
		wantErr     bool
	}{
		{
			name:        "zero horizon returns ErrInvalidHorizon",
			nodeID:      "node-1",
			horizonDays: 0,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
			},
			wantErr: true,
		},
		{
			name:        "db aggregates query error returns error",
			nodeID:      "node-1",
			horizonDays: 10,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-365*24*time.Hour), fixedTime, uint32(86400)).
					Return(nil, errDB)
			},
			wantErr: true,
		},
		{
			name:        "no aggregates skips prediction and returns nil",
			nodeID:      "node-1",
			horizonDays: 10,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-365*24*time.Hour), fixedTime, uint32(86400)).
					Return([]*protodatabasev1.ClimateAggregate{}, nil)
			},
			wantErr: false,
		},
		{
			name:        "inference rpc error returns error",
			nodeID:      "node-1",
			horizonDays: 10,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-365*24*time.Hour), fixedTime, uint32(86400)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedTime.Add(-24 * time.Hour))},
					}, nil)
				mockInf.EXPECT().
					PredictDailyWeather(gomock.Any(), gomock.Any()).
					Return(nil, errInf)
			},
			wantErr: true,
		},
		{
			name:        "store predictions error returns error",
			nodeID:      "node-1",
			horizonDays: 10,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-365*24*time.Hour), fixedTime, uint32(86400)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedTime.Add(-24 * time.Hour))},
					}, nil)
				mockInf.EXPECT().
					PredictDailyWeather(gomock.Any(), gomock.Any()).
					Return(&protoinferencev1.PredictDailyWeatherResponse{
						Predictions: []*protodatabasev1.DailyPrediction{
							{NodeId: "node-1", HorizonDays: 1},
						},
					}, nil)
				mockDB.EXPECT().
					StorePredictions(gomock.Any(), gomock.Len(1)).
					Return(errStore)
			},
			wantErr: true,
		},
		{
			name:        "successful daily forecast",
			nodeID:      "node-1",
			horizonDays: 10,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-365*24*time.Hour), fixedTime, uint32(86400)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedTime.Add(-24 * time.Hour))},
					}, nil)
				mockInf.EXPECT().
					PredictDailyWeather(gomock.Any(), gomock.AssignableToTypeOf(&protoinferencev1.PredictDailyWeatherRequest{})).
					Return(&protoinferencev1.PredictDailyWeatherResponse{
						Predictions: []*protodatabasev1.DailyPrediction{
							{NodeId: "node-1", HorizonDays: 1},
						},
					}, nil)
				mockDB.EXPECT().
					StorePredictions(gomock.Any(), gomock.Len(1)).
					Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockDB := mocks.NewMockDatabaseGateway(ctrl)
			mockInf := mocks.NewMockInferenceGateway(ctrl)
			tt.setup(mockDB, mockInf)

			c := &Controller{
				log:        logger,
				dbGateway:  mockDB,
				infGateway: mockInf,
				nowFunc:    func() time.Time { return fixedTime },
			}

			err := c.RunDailyForecastForNode(context.Background(), tt.nodeID, tt.horizonDays)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RunDailyForecastForNode() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunHourlyForecast(t *testing.T) {
	logger := zap.NewNop()
	fixedTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	errList := errors.New("list error")

	tests := []struct {
		name    string
		setup   func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway)
		wantErr bool
	}{
		{
			name: "list nodes fails",
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					ListNodeCoordinates(gomock.Any()).
					Return(nil, errList)
			},
			wantErr: true,
		},
		{
			name: "all nodes succeed",
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					ListNodeCoordinates(gomock.Any()).
					Return([]*protodatabasev1.NodeCoordinates{
						{NodeId: "node-1"},
						{NodeId: "node-2"},
					}, nil)

				for _, id := range []string{"node-1", "node-2"} {
					mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), id, fixedTime.Add(-40*24*time.Hour), fixedTime, uint32(3600)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedTime.Add(-time.Hour))},
					}, nil)
					mockInf.EXPECT().
						PredictHourlyWeather(gomock.Any(), gomock.Any()).
						Return(&protoinferencev1.PredictHourlyWeatherResponse{
							Predictions: []*protodatabasev1.HourlyPrediction{
								{NodeId: id, HorizonHours: 1},
							},
						}, nil)
					mockDB.EXPECT().
						StorePredictions(gomock.Any(), gomock.Len(1)).
						Return(nil)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockDB := mocks.NewMockDatabaseGateway(ctrl)
			mockInf := mocks.NewMockInferenceGateway(ctrl)
			tt.setup(mockDB, mockInf)

			c := &Controller{
				log:        logger,
				dbGateway:  mockDB,
				infGateway: mockInf,
				nowFunc:    func() time.Time { return fixedTime },
			}

			err := c.RunHourlyForecast(context.Background(), 24)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RunHourlyForecast() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunDailyForecast(t *testing.T) {
	logger := zap.NewNop()
	fixedTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	errList := errors.New("list error")

	tests := []struct {
		name    string
		setup   func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway)
		wantErr bool
	}{
		{
			name: "list nodes fails",
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					ListNodeCoordinates(gomock.Any()).
					Return(nil, errList)
			},
			wantErr: true,
		},
		{
			name: "all nodes succeed",
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					ListNodeCoordinates(gomock.Any()).
					Return([]*protodatabasev1.NodeCoordinates{
						{NodeId: "node-1"},
						{NodeId: "node-2"},
					}, nil)

				for _, id := range []string{"node-1", "node-2"} {
					mockDB.EXPECT().
						GetClimateAggregates(gomock.Any(), id, fixedTime.Add(-365*24*time.Hour), fixedTime, uint32(86400)).
						Return([]*protodatabasev1.ClimateAggregate{
							{BucketStart: timestamppb.New(fixedTime.Add(-24 * time.Hour))},
						}, nil)
					mockInf.EXPECT().
						PredictDailyWeather(gomock.Any(), gomock.Any()).
						Return(&protoinferencev1.PredictDailyWeatherResponse{
							Predictions: []*protodatabasev1.DailyPrediction{
								{NodeId: id, HorizonDays: 1},
							},
						}, nil)
					mockDB.EXPECT().
						StorePredictions(gomock.Any(), gomock.Len(1)).
						Return(nil)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockDB := mocks.NewMockDatabaseGateway(ctrl)
			mockInf := mocks.NewMockInferenceGateway(ctrl)
			tt.setup(mockDB, mockInf)

			c := &Controller{
				log:        logger,
				dbGateway:  mockDB,
				infGateway: mockInf,
				nowFunc:    func() time.Time { return fixedTime },
			}

			err := c.RunDailyForecast(context.Background(), 10)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RunDailyForecast() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
