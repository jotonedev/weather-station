-- Raw weather readings, one row per sample.
-- All measurement columns are nullable because not every node provides every field.
-- CONTRACT: node_id and ts are required (non-null) for every row; QuestDB does not
-- support NOT NULL column constraints, so this is enforced by the application layer.
CREATE TABLE IF NOT EXISTS weather (
    node_id        SYMBOL,
    ts             TIMESTAMP,
    wind_speed     FLOAT,
    wind_angle     FLOAT,
    raw_wind_speed FLOAT,
    raw_wind_angle FLOAT,
    temperature    FLOAT,
    humidity       FLOAT,
    pressure       FLOAT,
    rain_1h        FLOAT,
    rain_24h       FLOAT,
    rain_rate      FLOAT,
    pm1_0          INT,
    pm2_5          INT,
    pm10_0         INT,
    voc            INT,
    anemo_status          INT,
    anemo_agc             INT,
    anemo_agc_variance    INT,
    vane_status           INT,
    vane_agc              INT,
    vane_agc_variance     INT,
    rain_status           INT,
    rain_agc              INT,
    rain_agc_variance     INT
) TIMESTAMP(ts) PARTITION BY DAY;
