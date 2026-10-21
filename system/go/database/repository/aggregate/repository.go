package aggregate

import (
	"context"
	"fmt"
	"time"

	duckdb "weather-station/database/client"

	"go.uber.org/zap"
)

// Repository computes time-bucketed aggregates over the embedded DuckDB weather table.
type Repository struct {
	client *duckdb.Client
	log    *zap.Logger
}

func New(client *duckdb.Client, log *zap.Logger) *Repository {
	return &Repository{client: client, log: log}
}

// GetClimateAggregates returns temperature, humidity, pressure, rainfall and wind aggregates per time bucket for a node.
func (r *Repository) GetClimateAggregates(ctx context.Context, nodeID string, start, end time.Time, bucketSeconds uint32) ([]ClimateAggregate, error) {
	bucket := fmt.Sprintf("time_bucket(INTERVAL '%d seconds', ts)", bucketSeconds)
	query := "SELECT " + bucket + ` AS bucket_start,
		avg(temperature) AS avg_temperature_c,
		min(temperature) AS min_temperature_c,
		max(temperature) AS max_temperature_c,
		avg(humidity) AS avg_humidity_pct,
		min(humidity) AS min_humidity_pct,
		max(humidity) AS max_humidity_pct,
		avg(pressure) AS avg_pressure_hpa,
		min(pressure) AS min_pressure_hpa,
		max(pressure) AS max_pressure_hpa,
		avg(rain_1h) * (? / 3600.0) AS total_rainfall_mm,
		avg(wind_speed) AS avg_wind_speed_mps,
		fmod(degrees(atan2(avg(sin(radians(wind_angle))), avg(cos(radians(wind_angle))))) + 360.0, 360.0) AS avg_wind_direction_deg,
		count(*) AS sample_count
	FROM weather
	WHERE node_id = ? AND ts >= ? AND ts <= ?
	GROUP BY bucket_start
	ORDER BY bucket_start ASC`

	rows, err := r.client.DB().QueryContext(ctx, query, float64(bucketSeconds), nodeID, start, end)
	if err != nil {
		r.log.Error("failed to query climate aggregates", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var aggregates []ClimateAggregate
	for rows.Next() {
		var aggregate ClimateAggregate
		aggregate.NodeID = nodeID
		if err := rows.Scan(
			&aggregate.BucketStart,
			&aggregate.AvgTemperatureC, &aggregate.MinTemperatureC, &aggregate.MaxTemperatureC,
			&aggregate.AvgHumidityPct, &aggregate.MinHumidityPct, &aggregate.MaxHumidityPct,
			&aggregate.AvgPressureHpa, &aggregate.MinPressureHpa, &aggregate.MaxPressureHpa,
			&aggregate.TotalRainfallMm,
			&aggregate.AvgWindSpeedMps,
			&aggregate.AvgWindDirectionDeg,
			&aggregate.SampleCount,
		); err != nil {
			r.log.Error("failed to scan climate aggregate", zap.String("node_id", nodeID), zap.Error(err))
			return nil, err
		}
		aggregates = append(aggregates, aggregate)
	}
	if err := rows.Err(); err != nil {
		r.log.Error("error reading climate aggregate rows", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	return aggregates, nil
}

// GetAirQualityAggregates returns pm and voc aggregates per time bucket for a node.
func (r *Repository) GetAirQualityAggregates(ctx context.Context, nodeID string, start, end time.Time, bucketSeconds uint32) ([]AirQualityAggregate, error) {
	bucket := fmt.Sprintf("time_bucket(INTERVAL '%d seconds', ts)", bucketSeconds)
	query := "SELECT " + bucket + ` AS bucket_start,
		avg(pm1_0) AS avg_pm1,
		avg(pm2_5) AS avg_pm25,
		avg(pm10_0) AS avg_pm10,
		avg(voc) AS avg_voc,
		count(*) AS sample_count
	FROM weather
	WHERE node_id = ? AND ts >= ? AND ts <= ?
	GROUP BY bucket_start
	ORDER BY bucket_start ASC`

	rows, err := r.client.DB().QueryContext(ctx, query, nodeID, start, end)
	if err != nil {
		r.log.Error("failed to query air quality aggregates", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var aggregates []AirQualityAggregate
	for rows.Next() {
		var aggregate AirQualityAggregate
		aggregate.NodeID = nodeID
		if err := rows.Scan(
			&aggregate.BucketStart,
			&aggregate.AvgPm1, &aggregate.AvgPm25, &aggregate.AvgPm10, &aggregate.AvgVoc,
			&aggregate.SampleCount,
		); err != nil {
			r.log.Error("failed to scan air quality aggregate", zap.String("node_id", nodeID), zap.Error(err))
			return nil, err
		}
		aggregates = append(aggregates, aggregate)
	}
	if err := rows.Err(); err != nil {
		r.log.Error("error reading air quality aggregate rows", zap.String("node_id", nodeID), zap.Error(err))
		return nil, err
	}
	return aggregates, nil
}
