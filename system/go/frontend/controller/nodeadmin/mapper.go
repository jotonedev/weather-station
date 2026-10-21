package nodeadmin

import (
	"fmt"
	"strings"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func pbNodeToView(node *protodatabasev1.NodeCoordinates) NodeItemView {
	location := strings.TrimSpace(node.GetLocationName())
	if location == "" {
		location = "N/A"
	}
	return NodeItemView{
		NodeID:       node.GetNodeId(),
		LocationName: location,
		Latitude:     fmt.Sprintf("%.4f", node.GetLatitude()),
		Longitude:    fmt.Sprintf("%.4f", node.GetLongitude()),
		ElevationM:   fmt.Sprintf("%.1f m", node.GetElevationM()),
	}
}

func formInputToPB(input NodeFormInput) *protodatabasev1.NodeCoordinates {
	return &protodatabasev1.NodeCoordinates{
		NodeId:       strings.TrimSpace(input.NodeID),
		LocationName: strings.TrimSpace(input.LocationName),
		Latitude:     input.Latitude,
		Longitude:    input.Longitude,
		ElevationM:   input.ElevationM,
	}
}
