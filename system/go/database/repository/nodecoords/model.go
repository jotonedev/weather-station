package nodecoords

// NodeCoordinates describes a node's static physical location.
// NodeID, Latitude, Longitude and ElevationM are required (non-nullable).
type NodeCoordinates struct {
	NodeID       string
	Latitude     float64
	Longitude    float64
	ElevationM   float64
	LocationName string
}
