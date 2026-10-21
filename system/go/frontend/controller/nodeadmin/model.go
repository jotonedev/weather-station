package nodeadmin

// NodeFormInput represents user-submitted fields when creating or updating a node.
type NodeFormInput struct {
	NodeID       string
	LocationName string
	Latitude     float64
	Longitude    float64
	ElevationM   float64
}

// NodeItemView represents a registered weather station node for the admin UI.
type NodeItemView struct {
	NodeID       string
	LocationName string
	Latitude     string
	Longitude    string
	ElevationM   string
}

// PageData holds template context for the node administration page.
type PageData struct {
	Title         string
	Nodes         []NodeItemView
	StatusMessage string
	ErrorMessage  string
}
