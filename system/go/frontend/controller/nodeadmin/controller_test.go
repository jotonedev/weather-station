package nodeadmin

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"weather-station/frontend/gateway/database"
	"weather-station/frontend/internal/mocks"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestControllerSaveNode(t *testing.T) {
	errDB := errors.New("db failure")

	tests := []struct {
		name    string
		input   NodeFormInput
		setup   func(client *mocks.MockNodeDatabaseServiceClient)
		wantErr error
	}{
		{
			name: "valid node is saved",
			input: NodeFormInput{
				NodeID:       "station-alpha",
				LocationName: "Turin",
				Latitude:     45.07,
				Longitude:    7.68,
				ElevationM:   240,
			},
			setup: func(client *mocks.MockNodeDatabaseServiceClient) {
				client.EXPECT().
					CreateNode(gomock.Any(), &protodatabasev1.CreateNodeRequest{
						Node: &protodatabasev1.NodeCoordinates{
							NodeId:       "station-alpha",
							LocationName: "Turin",
							Latitude:     45.07,
							Longitude:    7.68,
							ElevationM:   240,
						},
					}).
					Return(&protodatabasev1.CreateNodeResponse{}, nil)
			},
		},
		{
			name: "missing node id returns validation error",
			input: NodeFormInput{
				NodeID:    "   ",
				Latitude:  45.0,
				Longitude: 7.0,
			},
			setup:   func(client *mocks.MockNodeDatabaseServiceClient) {},
			wantErr: ErrMissingNodeID,
		},
		{
			name: "invalid latitude returns validation error",
			input: NodeFormInput{
				NodeID:    "station-alpha",
				Latitude:  95.0,
				Longitude: 7.0,
			},
			setup:   func(client *mocks.MockNodeDatabaseServiceClient) {},
			wantErr: ErrInvalidLatitude,
		},
		{
			name: "invalid longitude returns validation error",
			input: NodeFormInput{
				NodeID:    "station-alpha",
				Latitude:  45.0,
				Longitude: -195.0,
			},
			setup:   func(client *mocks.MockNodeDatabaseServiceClient) {},
			wantErr: ErrInvalidLongitude,
		},
		{
			name: "database error is returned",
			input: NodeFormInput{
				NodeID:    "station-alpha",
				Latitude:  45.0,
				Longitude: 7.0,
			},
			setup: func(client *mocks.MockNodeDatabaseServiceClient) {
				client.EXPECT().
					CreateNode(gomock.Any(), gomock.Any()).
					Return(nil, errDB)
			},
			wantErr: errDB,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			nodeClient := mocks.NewMockNodeDatabaseServiceClient(ctrl)
			tc.setup(nodeClient)

			gw, err := database.New(database.GatewayParams{
				Logger:     zap.NewNop(),
				Config:     database.Config{Address: "127.0.0.1:8090"},
				NodeClient: nodeClient,
			})
			if err != nil {
				t.Fatalf("database.New() error = %v", err)
			}

			controller, err := New(ControllerParams{
				Logger:   zap.NewNop(),
				Database: gw,
			})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			err = controller.SaveNode(context.Background(), tc.input)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("SaveNode() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("SaveNode() unexpected error = %v", err)
			}
		})
	}
}

func TestControllerDeleteNodeAndRender(t *testing.T) {
	ctrl := gomock.NewController(t)
	nodeClient := mocks.NewMockNodeDatabaseServiceClient(ctrl)

	nodeClient.EXPECT().
		DeleteNode(gomock.Any(), &protodatabasev1.DeleteNodeRequest{NodeId: "station-alpha"}).
		Return(&protodatabasev1.DeleteNodeResponse{}, nil)

	nodeClient.EXPECT().
		ListNodeCoordinates(gomock.Any(), gomock.Any()).
		Return(&protodatabasev1.ListNodeCoordinatesResponse{
			Nodes: []*protodatabasev1.NodeCoordinates{
				{
					NodeId:       "station-beta",
					LocationName: "Valley",
					Latitude:     44.5,
					Longitude:    7.5,
					ElevationM:   120,
				},
			},
		}, nil)

	gw, err := database.New(database.GatewayParams{
		Logger:     zap.NewNop(),
		Config:     database.Config{Address: "127.0.0.1:8090"},
		NodeClient: nodeClient,
	})
	if err != nil {
		t.Fatalf("database.New() error = %v", err)
	}

	controller, err := New(ControllerParams{
		Logger:   zap.NewNop(),
		Database: gw,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := controller.DeleteNode(context.Background(), ""); !errors.Is(err, ErrMissingNodeID) {
		t.Fatalf("DeleteNode(\"\") error = %v, want %v", err, ErrMissingNodeID)
	}

	if err := controller.DeleteNode(context.Background(), "station-alpha"); err != nil {
		t.Fatalf("DeleteNode() unexpected error = %v", err)
	}

	rendered, err := controller.Render(context.Background(), "Node deleted", "")
	if err != nil {
		t.Fatalf("Render() unexpected error = %v", err)
	}
	html := string(rendered)
	if !strings.Contains(html, "station-beta") || !strings.Contains(html, "Node deleted") {
		t.Fatalf("rendered HTML missing expected content")
	}
}
