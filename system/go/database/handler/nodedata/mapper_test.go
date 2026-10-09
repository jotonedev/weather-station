package nodedata

import (
	"testing"

	"weather-station/database/repository/nodecoords"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestModelNodeToPB(t *testing.T) {
	node := nodecoords.NodeCoordinates{
		NodeID:       "n1",
		Latitude:     45.5,
		Longitude:    9.2,
		ElevationM:   120,
		LocationName: "roof",
	}

	response := modelNodeToPB(node)
	if response.NodeId != "n1" {
		t.Errorf("NodeId = %q, want n1", response.NodeId)
	}
	if response.Latitude != 45.5 || response.Longitude != 9.2 || response.ElevationM != 120 {
		t.Errorf("coords = (%v, %v, %v), want (45.5, 9.2, 120)", response.Latitude, response.Longitude, response.ElevationM)
	}
	if response.LocationName != "roof" {
		t.Errorf("LocationName = %q, want roof", response.LocationName)
	}
}

func TestPBNodeToModel(t *testing.T) {
	pbNode := &protodatabasev1.NodeCoordinates{
		NodeId:       "n1",
		Latitude:     45.5,
		Longitude:    9.2,
		ElevationM:   120,
		LocationName: "roof",
	}

	node := pbNodeToModel(pbNode)
	if node.NodeID != "n1" {
		t.Errorf("NodeID = %q, want n1", node.NodeID)
	}
	if node.Latitude != 45.5 || node.Longitude != 9.2 || node.ElevationM != 120 {
		t.Errorf("coords = (%v, %v, %v), want (45.5, 9.2, 120)", node.Latitude, node.Longitude, node.ElevationM)
	}
	if node.LocationName != "roof" {
		t.Errorf("LocationName = %q, want roof", node.LocationName)
	}
}
