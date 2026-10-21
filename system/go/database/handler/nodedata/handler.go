package nodedata

import (
	"context"

	controller "weather-station/database/controller/nodedata"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Handler struct {
	log *zap.Logger
	con *controller.Controller
	protodatabasev1.UnimplementedNodeDatabaseServiceServer
}

type HandlerParams struct {
	fx.In

	Logger     *zap.Logger
	Controller *controller.Controller
}

// New returns a handler for the NodeDatabaseService.
func New(params HandlerParams) *Handler {
	return &Handler{
		log: params.Logger,
		con: params.Controller,
	}
}

// Register exposes this handler as the NodeDatabaseService.
func (h *Handler) Register(s grpc.ServiceRegistrar) {
	protodatabasev1.RegisterNodeDatabaseServiceServer(s, h)
}

// CreateNode stores coordinates for a node.
func (h *Handler) CreateNode(ctx context.Context, req *protodatabasev1.CreateNodeRequest) (*protodatabasev1.CreateNodeResponse, error) {
	if err := h.con.CreateNode(ctx, pbNodeToModel(req.Node)); err != nil {
		return nil, err
	}
	return &protodatabasev1.CreateNodeResponse{}, nil
}

// UpdateNode updates the stored coordinates of a node.
func (h *Handler) UpdateNode(ctx context.Context, req *protodatabasev1.UpdateNodeRequest) (*protodatabasev1.UpdateNodeResponse, error) {
	if err := h.con.UpdateNode(ctx, pbNodeToModel(req.Node)); err != nil {
		return nil, err
	}
	return &protodatabasev1.UpdateNodeResponse{}, nil
}

// DeleteNode deletes a node and all of its associated readings and predictions.
func (h *Handler) DeleteNode(ctx context.Context, req *protodatabasev1.DeleteNodeRequest) (*protodatabasev1.DeleteNodeResponse, error) {
	if err := h.con.DeleteNode(ctx, req.GetNodeId()); err != nil {
		return nil, err
	}
	return &protodatabasev1.DeleteNodeResponse{}, nil
}

// GetNodeCoordinates returns coordinates for a node.
func (h *Handler) GetNodeCoordinates(ctx context.Context, req *protodatabasev1.GetNodeCoordinatesRequest) (*protodatabasev1.GetNodeCoordinatesResponse, error) {
	node, err := h.con.GetNodeCoordinates(ctx, req.NodeId)
	if err != nil {
		return nil, err
	}
	return &protodatabasev1.GetNodeCoordinatesResponse{Node: modelNodeToPB(node)}, nil
}

// ListNodeCoordinates returns coordinates for every node.
func (h *Handler) ListNodeCoordinates(ctx context.Context, req *protodatabasev1.ListNodeCoordinatesRequest) (*protodatabasev1.ListNodeCoordinatesResponse, error) {
	nodes, err := h.con.ListNodeCoordinates(ctx)
	if err != nil {
		return nil, err
	}
	response := &protodatabasev1.ListNodeCoordinatesResponse{Nodes: make([]*protodatabasev1.NodeCoordinates, 0, len(nodes))}
	for _, node := range nodes {
		response.Nodes = append(response.Nodes, modelNodeToPB(node))
	}
	return response, nil
}
