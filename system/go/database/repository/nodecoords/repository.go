package nodecoords

import (
	"context"
	"fmt"

	commonquestdb "weather-station/common/client/questdb"

	"go.uber.org/zap"
)

// Repository persists node coordinate metadata to the node_coordinates table.
type Repository struct {
	db  *commonquestdb.Client
	log *zap.Logger
}

func New(db *commonquestdb.Client, log *zap.Logger) *Repository {
	return &Repository{db: db, log: log}
}

// UpsertNodeCoordinates inserts or updates node coordinate rows keyed by node_id.
// Returns the number of rows affected.
func (r *Repository) UpsertNodeCoordinates(ctx context.Context, nodes []NodeCoordinates) (int64, error) {
	if len(nodes) == 0 {
		return 0, nil
	}

	var total int64
	// QuestDB upsert via INSERT ... ON CONFLICT keyed on the unique node_id column.
	const upsert = `INSERT INTO node_coordinates (node_id, latitude, longitude, elevation_m, location_name)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (node_id) DO UPDATE
		SET latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			elevation_m = EXCLUDED.elevation_m,
			location_name = EXCLUDED.location_name`

	for _, node := range nodes {
		tag, err := r.db.Exec(ctx, upsert, node.NodeID, node.Latitude, node.Longitude, node.ElevationM, node.LocationName)
		if err != nil {
			r.log.Error("failed to upsert node coordinates",
				zap.String("node_id", node.NodeID), zap.Error(err))
			return total, err
		}
		total += tag.RowsAffected()
	}
	r.log.Debug("upserted node coordinates", zap.Int64("nodes", total))
	return total, nil
}

// GetNodeCoordinates retrieves coordinates for a node.
func (r *Repository) GetNodeCoordinates(ctx context.Context, nodeID string) (*NodeCoordinates, error) {
	const query = `SELECT node_id, latitude, longitude, elevation_m, location_name
		FROM node_coordinates WHERE node_id = $1 LIMIT 1`

	rows, err := r.db.Query(ctx, query, nodeID)
	if err != nil {
		r.log.Error("failed to query node coordinates", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			r.log.Error("error reading node coordinates", zap.String("node_id", nodeID), zap.Error(err))
			return nil, err
		}
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	var node NodeCoordinates
	if err := rows.Scan(&node.NodeID, &node.Latitude, &node.Longitude, &node.ElevationM, &node.LocationName); err != nil {
		r.log.Error("failed to scan node coordinates", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	return &node, rows.Err()
}
