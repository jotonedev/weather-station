package nodecoords

import (
	"context"
	"fmt"

	duckdb "weather-station/database/client"

	"go.uber.org/zap"
)

// Repository persists node coordinate metadata to the embedded DuckDB node_coordinates table.
type Repository struct {
	client *duckdb.Client
	log    *zap.Logger
}

func New(client *duckdb.Client, log *zap.Logger) *Repository {
	return &Repository{client: client, log: log}
}

// UpsertNodeCoordinates inserts or updates node coordinate rows keyed by node_id and returns rows affected.
func (r *Repository) UpsertNodeCoordinates(ctx context.Context, nodes []NodeCoordinates) (int64, error) {
	if len(nodes) == 0 {
		return 0, nil
	}

	var total int64
	const upsert = `INSERT INTO node_coordinates (node_id, latitude, longitude, elevation_m, location_name)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (node_id) DO UPDATE
		SET latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			elevation_m = EXCLUDED.elevation_m,
			location_name = EXCLUDED.location_name;`

	for _, node := range nodes {
		res, err := r.client.DB().ExecContext(ctx, upsert, node.NodeID, node.Latitude, node.Longitude, node.ElevationM, node.LocationName)
		if err != nil {
			r.log.Error("failed to upsert node coordinates",
				zap.String("node_id", node.NodeID), zap.Error(err))
			return total, err
		}
		affected, _ := res.RowsAffected()
		total += affected
	}
	r.log.Debug("upserted node coordinates", zap.Int64("nodes", total))
	return total, nil
}

// GetNodeCoordinates retrieves coordinates for a node.
func (r *Repository) GetNodeCoordinates(ctx context.Context, nodeID string) (*NodeCoordinates, error) {
	const query = `SELECT node_id, latitude, longitude, elevation_m, location_name
		FROM node_coordinates WHERE node_id = ? LIMIT 1`

	rows, err := r.client.DB().QueryContext(ctx, query, nodeID)
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

// ListNodeCoordinates returns coordinates for every node in the table.
func (r *Repository) ListNodeCoordinates(ctx context.Context) ([]NodeCoordinates, error) {
	const query = `SELECT node_id, latitude, longitude, elevation_m, location_name
		FROM node_coordinates ORDER BY node_id ASC`

	rows, err := r.client.DB().QueryContext(ctx, query)
	if err != nil {
		r.log.Error("failed to query all node coordinates", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var nodes []NodeCoordinates
	for rows.Next() {
		var node NodeCoordinates
		if err := rows.Scan(&node.NodeID, &node.Latitude, &node.Longitude, &node.ElevationM, &node.LocationName); err != nil {
			r.log.Error("failed to scan node coordinates", zap.Error(err))
			return nil, err
		}
		nodes = append(nodes, node)
	}
	if err := rows.Err(); err != nil {
		r.log.Error("error reading node coordinate rows", zap.Error(err))
		return nil, err
	}
	return nodes, nil
}

// DeleteNode removes the node coordinate entry along with all readings and predictions for the node.
func (r *Repository) DeleteNode(ctx context.Context, nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("node_id is required")
	}

	tx, err := r.client.DB().BeginTx(ctx, nil)
	if err != nil {
		r.log.Error("failed to start transaction for node deletion", zap.String("node_id", nodeID), zap.Error(err))
		return err
	}
	defer tx.Rollback()

	statements := []string{
		`DELETE FROM weather WHERE node_id = ?;`,
		`DELETE FROM hourly_predictions WHERE node_id = ?;`,
		`DELETE FROM daily_predictions WHERE node_id = ?;`,
		`DELETE FROM node_coordinates WHERE node_id = ?;`,
	}

	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt, nodeID); err != nil {
			r.log.Error("failed to delete node rows", zap.String("node_id", nodeID), zap.Error(err))
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		r.log.Error("failed to commit node deletion transaction", zap.String("node_id", nodeID), zap.Error(err))
		return err
	}

	r.log.Debug("deleted node and associated records", zap.String("node_id", nodeID))
	return nil
}
