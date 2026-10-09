-- Forecast predictions: only rain, temperature, humidity and wind direction
-- are predicted. Rows are keyed by issue time (ts) plus the forecast horizon.
-- CONTRACT: node_id, ts and horizon_hours are required (non-null); QuestDB does
-- not support NOT NULL column constraints, so this is enforced by the application layer.
CREATE TABLE IF NOT EXISTS predictions (
    node_id           SYMBOL,
    ts                TIMESTAMP,
    horizon_hours     INT,
    rainfall_mm       DOUBLE,
    temperature_c     DOUBLE,
    humidity_pct      DOUBLE,
    wind_direction_deg DOUBLE
) TIMESTAMP(ts) PARTITION BY DAY;
