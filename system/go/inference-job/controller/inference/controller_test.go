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

func TestRunForecastForNode(t *testing.T) {
	logger := zap.NewNop()
	fixedTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	errDB := errors.New("db failed")
	errInf := errors.New("inference failed")
	errStore := errors.New("store failed")

	tests := []struct {
		name    string
		nodeID  string
		mode    protoinferencev1.PredictionMode
		setup   func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway)
		wantErr bool
	}{
		{
			name:   "unspecified mode returns ErrInvalidMode",
			nodeID: "node-1",
			mode:   protoinferencev1.PredictionMode_PREDICTION_MODE_UNSPECIFIED,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				// No calls expected.
			},
			wantErr: true,
		},
		{
			name:   "db aggregates query error returns error",
			nodeID: "node-1",
			mode:   protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_24_HOURS,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-40*24*time.Hour), fixedTime, uint32(3600)).
					Return(nil, errDB)
			},
			wantErr: true,
		},
		{
			name:   "no aggregates skips prediction and returns nil",
			nodeID: "node-1",
			mode:   protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_24_HOURS,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-40*24*time.Hour), fixedTime, uint32(3600)).
					Return([]*protodatabasev1.ClimateAggregate{}, nil)
			},
			wantErr: false,
		},
		{
			name:   "inference rpc error returns error",
			nodeID: "node-1",
			mode:   protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_24_HOURS,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-40*24*time.Hour), fixedTime, uint32(3600)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedTime.Add(-time.Hour))},
					}, nil)
				mockInf.EXPECT().
					PredictWeather(gomock.Any(), gomock.Any()).
					Return(nil, errInf)
			},
			wantErr: true,
		},
		{
			name:   "store predictions error returns error",
			nodeID: "node-1",
			mode:   protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_24_HOURS,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-40*24*time.Hour), fixedTime, uint32(3600)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedTime.Add(-time.Hour))},
					}, nil)
				mockInf.EXPECT().
					PredictWeather(gomock.Any(), gomock.Any()).
					Return(&protoinferencev1.PredictWeatherResponse{
						Predictions: []*protodatabasev1.Prediction{
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
			name:   "successful 24h mode forecast",
			nodeID: "node-1",
			mode:   protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_24_HOURS,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-40*24*time.Hour), fixedTime, uint32(3600)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedTime.Add(-time.Hour))},
					}, nil)
				mockInf.EXPECT().
					PredictWeather(gomock.Any(), gomock.AssignableToTypeOf(&protoinferencev1.PredictWeatherRequest{})).
					Return(&protoinferencev1.PredictWeatherResponse{
						Predictions: []*protodatabasev1.Prediction{
							{NodeId: "node-1", HorizonHours: 1},
						},
					}, nil)
				mockDB.EXPECT().
					StorePredictions(gomock.Any(), gomock.Len(1)).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "successful 10-day mode forecast",
			nodeID: "node-1",
			mode:   protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_10_DAYS,
			setup: func(mockDB *mocks.MockDatabaseGateway, mockInf *mocks.MockInferenceGateway) {
				mockDB.EXPECT().
					GetClimateAggregates(gomock.Any(), "node-1", fixedTime.Add(-365*24*time.Hour), fixedTime, uint32(86400)).
					Return([]*protodatabasev1.ClimateAggregate{
						{BucketStart: timestamppb.New(fixedTime.Add(-24 * time.Hour))},
					}, nil)
				mockInf.EXPECT().
					PredictWeather(gomock.Any(), gomock.AssignableToTypeOf(&protoinferencev1.PredictWeatherRequest{})).
					Return(&protoinferencev1.PredictWeatherResponse{
						Predictions: []*protodatabasev1.Prediction{
							{NodeId: "node-1", HorizonHours: 24},
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

			err := c.RunForecastForNode(context.Background(), tt.nodeID, tt.mode)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RunForecastForNode() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunForecast(t *testing.T) {
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
						PredictWeather(gomock.Any(), gomock.Any()).
						Return(&protoinferencev1.PredictWeatherResponse{
							Predictions: []*protodatabasev1.Prediction{
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

			err := c.RunForecast(context.Background(), protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_24_HOURS)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RunForecast() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
