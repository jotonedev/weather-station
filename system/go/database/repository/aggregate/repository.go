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
	var query string
	var args []any

	if bucketSeconds >= 3600 && bucketSeconds%3600 == 0 {
		bucket := fmt.Sprintf("time_bucket(INTERVAL '%d seconds', ts)", bucketSeconds)
		outerBucket := fmt.Sprintf("time_bucket(INTERVAL '%d seconds', hour_start)", bucketSeconds)
		query = fmt.Sprintf(`WITH base AS (
			SELECT %s AS bucket_start,
				avg(temperature) AS avg_temperature_c,
				min(temperature) AS min_temperature_c,
				max(temperature) AS max_temperature_c,
				avg(humidity) AS avg_humidity_pct,
				min(humidity) AS min_humidity_pct,
				max(humidity) AS max_humidity_pct,
				avg(pressure) AS avg_pressure_hpa,
				min(pressure) AS min_pressure_hpa,
				max(pressure) AS max_pressure_hpa,
				avg(wind_speed) AS avg_wind_speed_mps,
				fmod(degrees(atan2(avg(sin(radians(wind_angle))), avg(cos(radians(wind_angle))))) + 360.0, 360.0) AS avg_wind_direction_deg,
				max(wind_speed) AS max_wind_speed_mps,
				avg(-wind_speed * sin(radians(wind_angle))) AS avg_wind_u_mps,
				avg(-wind_speed * cos(radians(wind_angle))) AS avg_wind_v_mps,
				max(rain_rate) AS max_rain_rate_mmh,
				count(*) AS sample_count
			FROM weather
			WHERE node_id = ? AND ts >= ? AND ts < ?
			GROUP BY bucket_start
		),
		hourly_rain AS (
			SELECT %s AS bucket_start,
				sum(hour_rainfall_mm) AS total_rainfall_mm
			FROM (
				SELECT time_bucket(INTERVAL '3600 seconds', ts) AS hour_start,
					arg_max(rain_1h, ts) AS hour_rainfall_mm
				FROM weather
				WHERE node_id = ? AND ts >= ? AND ts < ? AND rain_1h IS NOT NULL
				GROUP BY hour_start
			)
			GROUP BY bucket_start
		)
		SELECT b.bucket_start,
			b.avg_temperature_c, b.min_temperature_c, b.max_temperature_c,
			b.avg_humidity_pct, b.min_humidity_pct, b.max_humidity_pct,
			b.avg_pressure_hpa, b.min_pressure_hpa, b.max_pressure_hpa,
			hr.total_rainfall_mm,
			b.avg_wind_speed_mps, b.avg_wind_direction_deg, b.max_wind_speed_mps,
			b.avg_wind_u_mps, b.avg_wind_v_mps,
			b.max_rain_rate_mmh,
			b.sample_count
		FROM base b
		LEFT JOIN hourly_rain hr ON b.bucket_start = hr.bucket_start
		ORDER BY b.bucket_start ASC`, bucket, outerBucket)
		args = []any{nodeID, start, end, nodeID, start, end}
	} else {
		bucket := fmt.Sprintf("time_bucket(INTERVAL '%d seconds', ts)", bucketSeconds)
		query = "SELECT " + bucket + ` AS bucket_start,
			avg(temperature) AS avg_temperature_c,
			min(temperature) AS min_temperature_c,
			max(temperature) AS max_temperature_c,
			avg(humidity) AS avg_humidity_pct,
			min(humidity) AS min_humidity_pct,
			max(humidity) AS max_humidity_pct,
			avg(pressure) AS avg_pressure_hpa,
			min(pressure) AS min_pressure_hpa,
			max(pressure) AS max_pressure_hpa,
			coalesce(avg(rain_rate) * (? / 3600.0), avg(rain_1h) * (? / 3600.0)) AS total_rainfall_mm,
			avg(wind_speed) AS avg_wind_speed_mps,
			fmod(degrees(atan2(avg(sin(radians(wind_angle))), avg(cos(radians(wind_angle))))) + 360.0, 360.0) AS avg_wind_direction_deg,
			max(wind_speed) AS max_wind_speed_mps,
			avg(-wind_speed * sin(radians(wind_angle))) AS avg_wind_u_mps,
			avg(-wind_speed * cos(radians(wind_angle))) AS avg_wind_v_mps,
			max(rain_rate) AS max_rain_rate_mmh,
			count(*) AS sample_count
		FROM weather
		WHERE node_id = ? AND ts >= ? AND ts < ?
		GROUP BY bucket_start
		ORDER BY bucket_start ASC`
		bucketSecFloat := float64(bucketSeconds)
		args = []any{bucketSecFloat, bucketSecFloat, nodeID, start, end}
	}

	rows, err := r.client.DB().QueryContext(ctx, query, args...)
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
			&aggregate.MaxWindSpeedMps,
			&aggregate.AvgWindUMps,
			&aggregate.AvgWindVMps,
			&aggregate.MaxRainRateMmh,
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

// GetAirQualityAggregates returns pm aggregates per time bucket for a node.
func (r *Repository) GetAirQualityAggregates(ctx context.Context, nodeID string, start, end time.Time, bucketSeconds uint32) ([]AirQualityAggregate, error) {
	bucket := fmt.Sprintf("time_bucket(INTERVAL '%d seconds', ts)", bucketSeconds)
	query := "SELECT " + bucket + ` AS bucket_start,
		avg(pm1_0) AS avg_pm1,
		avg(pm2_5) AS avg_pm25,
		avg(pm10_0) AS avg_pm10,
		count(*) AS sample_count
	FROM weather
	WHERE node_id = ? AND ts >= ? AND ts < ?
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
			&aggregate.AvgPm1, &aggregate.AvgPm25, &aggregate.AvgPm10,
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
