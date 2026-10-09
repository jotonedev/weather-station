package nodedata

import (
	"weather-station/database/repository/nodecoords"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

// pbNodeToModel converts a proto NodeCoordinates into the repository model.
func pbNodeToModel(node *protodatabasev1.NodeCoordinates) nodecoords.NodeCoordinates {
	return nodecoords.NodeCoordinates{
		NodeID:       node.NodeId,
		Latitude:     node.Latitude,
		Longitude:    node.Longitude,
		ElevationM:   node.ElevationM,
		LocationName: node.LocationName,
	}
}

// modelNodeToPB converts a repository NodeCoordinates model into proto NodeCoordinates.
func modelNodeToPB(node nodecoords.NodeCoordinates) *protodatabasev1.NodeCoordinates {
	return &protodatabasev1.NodeCoordinates{
		NodeId:       node.NodeID,
		Latitude:     node.Latitude,
		Longitude:    node.Longitude,
		ElevationM:   node.ElevationM,
		LocationName: node.LocationName,
	}
}
