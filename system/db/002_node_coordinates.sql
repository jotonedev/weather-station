-- Static node location metadata, upserted by node_id.
-- CONTRACT: node_id, latitude, longitude and elevation_m are required (non-null);
-- location_name is optional. QuestDB does not support NOT NULL column constraints,
-- so this is enforced by the application layer.
CREATE TABLE IF NOT EXISTS node_coordinates (
    node_id       STRING,
    latitude      DOUBLE,
    longitude     DOUBLE,
    elevation_m   DOUBLE,
    location_name STRING
);
