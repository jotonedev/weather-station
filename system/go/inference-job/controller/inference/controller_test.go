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
			if tt.wantErr && !errors.Is(err, ErrNoGateways) {
				t.Errorf("New() error = %v, want ErrNoGateways", err)
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
		now          time.Time
		setup        func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway)
		wantErr      bool
		wantErrIs    error
	}{
		{
			name:         "zero horizon returns ErrInvalidHorizon",
			nodeID:       "node-1",
			horizonHours: 0,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
			},
			wantErr:   true,
			wantErrIs: ErrInvalidHorizon,
		},
		{
			name:         "negative horizon returns ErrInvalidHorizon",
			nodeID:       "node-1",
			horizonHours: -1,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
			},
			wantErr:   true,
			wantErrIs: ErrInvalidHorizon,
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
			wantErr:   true,
			wantErrIs: errDB,
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
			wantErr:   true,
			wantErrIs: errInf,
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
					StoreHourlyPredictions(gomock.Any(), gomock.Len(1)).
					Return(errStore)
			},
			wantErr:   true,
			wantErrIs: errStore,
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
					StoreHourlyPredictions(gomock.Any(), gomock.Len(1)).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:         "hourly forecast with non-zero minute and CET timezone truncates to :00 and excludes in-progress bucket",
			nodeID:       "node-1",
			horizonHours: 24,
			now:          time.Date(2026, 10, 10, 13, 45, 24, 0, time.FixedZone("CET", 3600)),
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				// 13:45:24 CET = 12:45:24 UTC -> truncated to 12:00:00 UTC (fixedTime)
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-40*24*time.Hour), fixedTime, uint32(3600)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedTime.Add(-time.Hour))},
						{BucketStart: timestamppb.New(fixedTime)},
					}, nil)
				mockInf.EXPECT().
					PredictHourlyWeather(gomock.Any(), gomock.AssignableToTypeOf(&protoinferencev1.PredictHourlyWeatherRequest{})).
					DoAndReturn(func(_ context.Context, req *protoinferencev1.PredictHourlyWeatherRequest) (*protoinferencev1.PredictHourlyWeatherResponse, error) {
						if !req.IssueTime.AsTime().Equal(fixedTime) {
							t.Fatalf("expected IssueTime %v, got %v", fixedTime, req.IssueTime.AsTime())
						}
						if len(req.History) != 1 {
							t.Fatalf("expected 1 history item (excluding in-progress bucket), got %d", len(req.History))
						}
						return &protoinferencev1.PredictHourlyWeatherResponse{
							Predictions: []*protodatabasev1.HourlyPrediction{
								{NodeId: "node-1", HorizonHours: 1},
							},
						}, nil
					})
				mockDB.EXPECT().
					StoreHourlyPredictions(gomock.Any(), gomock.Len(1)).
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

			currentNow := fixedTime
			if !tt.now.IsZero() {
				currentNow = tt.now
			}

			c := &Controller{
				log:        logger,
				dbGateway:  mockDB,
				infGateway: mockInf,
				nowFunc:    func() time.Time { return currentNow },
			}

			err := c.RunHourlyForecastForNode(context.Background(), tt.nodeID, tt.horizonHours)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RunHourlyForecastForNode() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("RunHourlyForecastForNode() error = %v, want %v", err, tt.wantErrIs)
			}
		})
	}
}

func TestRunDailyForecastForNode(t *testing.T) {
	logger := zap.NewNop()
	fixedTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	fixedGridEnd := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	errDB := errors.New("db failed")
	errInf := errors.New("inference failed")
	errStore := errors.New("store failed")

	tests := []struct {
		name        string
		nodeID      string
		horizonDays int32
		now         time.Time
		setup       func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway)
		wantErr     bool
		wantErrIs   error
	}{
		{
			name:        "zero horizon returns ErrInvalidHorizon",
			nodeID:      "node-1",
			horizonDays: 0,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
			},
			wantErr:   true,
			wantErrIs: ErrInvalidHorizon,
		},
		{
			name:        "db aggregates query error returns error",
			nodeID:      "node-1",
			horizonDays: 10,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedGridEnd.Add(-400*24*time.Hour), fixedGridEnd, uint32(10800)).
					Return(nil, errDB)
			},
			wantErr:   true,
			wantErrIs: errDB,
		},
		{
			name:        "no aggregates skips prediction and returns nil",
			nodeID:      "node-1",
			horizonDays: 10,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedGridEnd.Add(-400*24*time.Hour), fixedGridEnd, uint32(10800)).
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
					GetClimateAggregates(gomock.Any(), "node-1", fixedGridEnd.Add(-400*24*time.Hour), fixedGridEnd, uint32(10800)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedGridEnd.Add(-3 * time.Hour))},
					}, nil)
				mockInf.EXPECT().
					PredictDailyWeather(gomock.Any(), gomock.Any()).
					Return(nil, errInf)
			},
			wantErr:   true,
			wantErrIs: errInf,
		},
		{
			name:        "store predictions error returns error",
			nodeID:      "node-1",
			horizonDays: 10,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedGridEnd.Add(-400*24*time.Hour), fixedGridEnd, uint32(10800)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedGridEnd.Add(-3 * time.Hour))},
					}, nil)
				mockInf.EXPECT().
					PredictDailyWeather(gomock.Any(), gomock.Any()).
					Return(&protoinferencev1.PredictDailyWeatherResponse{
						Predictions: []*protodatabasev1.DailyPrediction{
							{NodeId: "node-1", HorizonDays: 0},
						},
					}, nil)
				mockDB.EXPECT().
					StoreDailyPredictions(gomock.Any(), gomock.Len(1)).
					Return(errStore)
			},
			wantErr:   true,
			wantErrIs: errStore,
		},
		{
			name:        "successful daily forecast",
			nodeID:      "node-1",
			horizonDays: 10,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedGridEnd.Add(-400*24*time.Hour), fixedGridEnd, uint32(10800)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedGridEnd.Add(-3 * time.Hour))},
					}, nil)
				mockInf.EXPECT().
					PredictDailyWeather(gomock.Any(), gomock.AssignableToTypeOf(&protoinferencev1.PredictDailyWeatherRequest{})).
					Return(&protoinferencev1.PredictDailyWeatherResponse{
						Predictions: []*protodatabasev1.DailyPrediction{
							{NodeId: "node-1", HorizonDays: 0},
						},
					}, nil)
				mockDB.EXPECT().
					StoreDailyPredictions(gomock.Any(), gomock.Len(1)).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "daily forecast aligns gridEnd to 3h boundary and issueTime to 1h boundary",
			nodeID:      "node-1",
			horizonDays: 10,
			now:         time.Date(2026, 10, 10, 14, 45, 24, 0, time.FixedZone("CEST", 7200)),
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				// 14:45:24 CEST = 12:45:24 UTC -> issueTime 12:00 UTC, gridEnd 12:00 UTC
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedGridEnd.Add(-400*24*time.Hour), fixedGridEnd, uint32(10800)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC))},
						{BucketStart: timestamppb.New(time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC))},
					}, nil)
				mockInf.EXPECT().
					PredictDailyWeather(gomock.Any(), gomock.AssignableToTypeOf(&protoinferencev1.PredictDailyWeatherRequest{})).
					DoAndReturn(func(_ context.Context, req *protoinferencev1.PredictDailyWeatherRequest) (*protoinferencev1.PredictDailyWeatherResponse, error) {
						if !req.IssueTime.AsTime().Equal(fixedTime) {
							t.Fatalf("expected IssueTime %v, got %v", fixedTime, req.IssueTime.AsTime())
						}
						if len(req.History) != 1 {
							t.Fatalf("expected 1 history item excluding unfinished 3h bucket, got %d", len(req.History))
						}
						return &protoinferencev1.PredictDailyWeatherResponse{
							Predictions: []*protodatabasev1.DailyPrediction{
								{NodeId: "node-1", HorizonDays: 0},
							},
						}, nil
					})
				mockDB.EXPECT().
					StoreDailyPredictions(gomock.Any(), gomock.Len(1)).
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

			currentNow := fixedTime
			if !tt.now.IsZero() {
				currentNow = tt.now
			}

			c := &Controller{
				log:        logger,
				dbGateway:  mockDB,
				infGateway: mockInf,
				nowFunc:    func() time.Time { return currentNow },
			}

			err := c.RunDailyForecastForNode(context.Background(), tt.nodeID, tt.horizonDays)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RunDailyForecastForNode() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("RunDailyForecastForNode() error = %v, want %v", err, tt.wantErrIs)
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
						StoreHourlyPredictions(gomock.Any(), gomock.Len(1)).
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
			if tt.wantErr && !errors.Is(err, errList) {
				t.Errorf("RunHourlyForecast() error = %v, want %v", err, errList)
			}
		})
	}
}

func TestRunDailyForecast(t *testing.T) {
	logger := zap.NewNop()
	fixedTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	fixedGridEnd := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
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
						GetClimateAggregates(gomock.Any(), id, fixedGridEnd.Add(-400*24*time.Hour), fixedGridEnd, uint32(10800)).
						Return([]*protodatabasev1.ClimateAggregate{
							{BucketStart: timestamppb.New(fixedGridEnd.Add(-3 * time.Hour))},
						}, nil)
					mockInf.EXPECT().
						PredictDailyWeather(gomock.Any(), gomock.Any()).
						Return(&protoinferencev1.PredictDailyWeatherResponse{
							Predictions: []*protodatabasev1.DailyPrediction{
								{NodeId: id, HorizonDays: 0},
							},
						}, nil)
					mockDB.EXPECT().
						StoreDailyPredictions(gomock.Any(), gomock.Len(1)).
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
			if tt.wantErr && !errors.Is(err, errList) {
				t.Errorf("RunDailyForecast() error = %v, want %v", err, errList)
			}
		})
	}
}

func TestRunHourlyThenDailyBlendsTodayAndKeepsUniformIssueTime(t *testing.T) {
	logger := zap.NewNop()
	fixedTime := time.Date(2026, 10, 10, 8, 25, 0, 0, time.UTC)
	expectedIssue := time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
	expectedGridEnd := time.Date(2026, 10, 10, 6, 0, 0, 0, time.UTC)

	ctrl := gomock.NewController(t)
	mockDB := mocks.NewMockDatabaseGateway(ctrl)
	mockInf := mocks.NewMockInferenceGateway(ctrl)

	rain2mm := float32(2.0)
	temp6 := float32(6.0)
	temp14 := float32(14.0)
	probLow := float32(0.45)

	mockDB.EXPECT().
		GetClimateAggregates(gomock.Any(), "node-1", expectedIssue.Add(-40*24*time.Hour), expectedIssue, uint32(3600)).
		Return([]*protodatabasev1.ClimateAggregate{
			{
				BucketStart:     timestamppb.New(time.Date(2026, 10, 10, 5, 0, 0, 0, time.UTC)),
				MinTemperatureC: &temp6,
				MaxTemperatureC: &temp6,
				TotalRainfallMm: &rain2mm,
			},
		}, nil)
	mockInf.EXPECT().
		PredictHourlyWeather(gomock.Any(), gomock.Any()).
		Return(&protoinferencev1.PredictHourlyWeatherResponse{
			Predictions: []*protodatabasev1.HourlyPrediction{
				{NodeId: "node-1", HorizonHours: 1, TemperatureC: &temp14},
			},
		}, nil)
	mockDB.EXPECT().StoreHourlyPredictions(gomock.Any(), gomock.Len(1)).Return(nil)

	mockDB.EXPECT().
		GetClimateAggregates(gomock.Any(), "node-1", expectedGridEnd.Add(-400*24*time.Hour), expectedGridEnd, uint32(10800)).
		Return([]*protodatabasev1.ClimateAggregate{
			{BucketStart: timestamppb.New(expectedGridEnd.Add(-3 * time.Hour))},
		}, nil)
	mockInf.EXPECT().
		PredictDailyWeather(gomock.Any(), gomock.Any()).
		Return(&protoinferencev1.PredictDailyWeatherResponse{
			Predictions: []*protodatabasev1.DailyPrediction{
				{NodeId: "node-1", HorizonDays: 0, RainProbability: &probLow},
				{NodeId: "node-1", HorizonDays: 1, RainProbability: &probLow},
			},
		}, nil)
	mockDB.EXPECT().
		StoreDailyPredictions(gomock.Any(), gomock.Len(2)).
		DoAndReturn(func(_ context.Context, preds []*protodatabasev1.DailyPrediction) error {
			if preds[0].GetRainProbability() != 1.0 {
				t.Fatalf("Today RainProbability = %v, want 1.0", preds[0].GetRainProbability())
			}
			if preds[0].GetMinTemperatureC() != 6.0 || preds[0].GetMaxTemperatureC() != 14.0 {
				t.Fatalf("Today temps = (%v, %v), want (6, 14)", preds[0].GetMinTemperatureC(), preds[0].GetMaxTemperatureC())
			}
			if !preds[0].GetIssueTime().AsTime().Equal(expectedIssue) || !preds[1].GetIssueTime().AsTime().Equal(expectedIssue) {
				t.Fatalf("IssueTimes not uniform at %v", expectedIssue)
			}
			return nil
		})

	c := &Controller{
		log:        logger,
		dbGateway:  mockDB,
		infGateway: mockInf,
		nowFunc:    func() time.Time { return fixedTime },
		intraday:   make(map[string]intradaySnapshot),
	}

	if err := c.RunHourlyForecastForNode(context.Background(), "node-1", 24); err != nil {
		t.Fatalf("RunHourlyForecastForNode() error = %v", err)
	}
	if err := c.RunDailyForecastForNode(context.Background(), "node-1", 10); err != nil {
		t.Fatalf("RunDailyForecastForNode() error = %v", err)
	}
}
