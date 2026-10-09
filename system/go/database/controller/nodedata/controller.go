package nodedata

import (
	"context"

	"weather-station/database/repository/nodecoords"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Controller holds node coordinate business logic.
type Controller struct {
	log  *zap.Logger
	repo *nodecoords.Repository
}

type ControllerParams struct {
	fx.In

	Logger *zap.Logger
	Repo   *nodecoords.Repository
}

func New(params ControllerParams) (*Controller, error) {
	return &Controller{
		log:  params.Logger,
		repo: params.Repo,
	}, nil
}

// CreateNode stores coordinates for a new node.
func (c *Controller) CreateNode(ctx context.Context, node nodecoords.NodeCoordinates) error {
	_, err := c.repo.UpsertNodeCoordinates(ctx, []nodecoords.NodeCoordinates{node})
	return err
}

// UpdateNode updates the stored coordinates of an existing node.
func (c *Controller) UpdateNode(ctx context.Context, node nodecoords.NodeCoordinates) error {
	_, err := c.repo.UpsertNodeCoordinates(ctx, []nodecoords.NodeCoordinates{node})
	return err
}

// GetNodeCoordinates returns the stored coordinates for a node.
func (c *Controller) GetNodeCoordinates(ctx context.Context, nodeID string) (nodecoords.NodeCoordinates, error) {
	node, err := c.repo.GetNodeCoordinates(ctx, nodeID)
	if err != nil {
		return nodecoords.NodeCoordinates{}, err
	}
	return *node, nil
}

// ListNodeCoordinates returns the stored coordinates for every node.
func (c *Controller) ListNodeCoordinates(ctx context.Context) ([]nodecoords.NodeCoordinates, error) {
	return c.repo.ListNodeCoordinates(ctx)
}
