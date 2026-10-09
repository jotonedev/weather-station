-- Forecast predictions: only rain, temperature, humidity and wind direction
-- are predicted. Each metric carries a confidence in the range 0 to 1.
-- Rows are keyed by issue time (ts) plus the forecast horizon.
-- CONTRACT: node_id, ts and horizon_hours are required (non-null); QuestDB does
-- not support NOT NULL column constraints, so this is enforced by the application layer.
CREATE TABLE IF NOT EXISTS predictions (
    node_id           SYMBOL,
    ts                TIMESTAMP,
    horizon_hours     INT,
    rainfall_mm       FLOAT,
    temperature_c     FLOAT,
    humidity_pct      FLOAT,
    wind_direction_deg FLOAT,
    rainfall_confidence       FLOAT,
    temperature_confidence    FLOAT,
    humidity_confidence       FLOAT,
    wind_direction_confidence FLOAT
) TIMESTAMP(ts) PARTITION BY DAY;
