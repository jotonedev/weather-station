package prediction

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	duckdb "weather-station/database/client"

	"go.uber.org/zap"
)

type hourlyBatch struct {
	ts          time.Time
	predictions []HourlyPrediction
}

type dailyBatch struct {
	ts          time.Time
	predictions []DailyPrediction
}

var (
	reposMu sync.RWMutex
	repos   = make(map[*Repository]struct{})
)

// Repository stores the latest hourly and daily forecast predictions per node in memory.
type Repository struct {
	log    *zap.Logger
	mu     sync.RWMutex
	hourly map[string]hourlyBatch
	daily  map[string]dailyBatch
}

func New(_ *duckdb.Client, log *zap.Logger) *Repository {
	r := &Repository{
		log:    log,
		hourly: make(map[string]hourlyBatch),
		daily:  make(map[string]dailyBatch),
	}
	reposMu.Lock()
	repos[r] = struct{}{}
	reposMu.Unlock()
	return r
}

// StoreHourlyPredictions stores the latest hourly prediction batch per node in memory and returns the number of items processed.
func (r *Repository) StoreHourlyPredictions(_ context.Context, predictions []HourlyPrediction) (int64, error) {
	if len(predictions) == 0 {
		return 0, nil
	}

	latestTs := make(map[string]time.Time)
	for _, pred := range predictions {
		if cur, ok := latestTs[pred.NodeID]; !ok || pred.Ts.After(cur) {
			latestTs[pred.NodeID] = pred.Ts
		}
	}

	byNode := make(map[string]map[int32]HourlyPrediction, len(latestTs))
	for _, pred := range predictions {
		if !pred.Ts.Equal(latestTs[pred.NodeID]) {
			continue
		}
		horizons := byNode[pred.NodeID]
		if horizons == nil {
			horizons = make(map[int32]HourlyPrediction)
			byNode[pred.NodeID] = horizons
		}
		horizons[pred.HorizonHours] = pred
	}

	batches := make(map[string]hourlyBatch, len(byNode))
	for nodeID, horizons := range byNode {
		items := make([]HourlyPrediction, 0, len(horizons))
		for _, pred := range horizons {
			items = append(items, pred)
		}
		slices.SortFunc(items, func(a, b HourlyPrediction) int {
			return cmp.Compare(a.HorizonHours, b.HorizonHours)
		})
		batches[nodeID] = hourlyBatch{
			ts:          latestTs[nodeID],
			predictions: items,
		}
	}

	r.mu.Lock()
	if r.hourly == nil {
		r.hourly = make(map[string]hourlyBatch)
	}
	for nodeID, batch := range batches {
		if existing, ok := r.hourly[nodeID]; !ok || !batch.ts.Before(existing.ts) {
			r.hourly[nodeID] = batch
		}
	}
	r.mu.Unlock()

	stored := int64(len(predictions))
	if r.log != nil {
		r.log.Debug("stored hourly predictions", zap.Int64("stored", stored))
	}
	return stored, nil
}

// StoreDailyPredictions stores the latest daily prediction batch per node in memory and returns the number of items processed.
func (r *Repository) StoreDailyPredictions(_ context.Context, predictions []DailyPrediction) (int64, error) {
	if len(predictions) == 0 {
		return 0, nil
	}

	latestTs := make(map[string]time.Time)
	for _, pred := range predictions {
		if cur, ok := latestTs[pred.NodeID]; !ok || pred.Ts.After(cur) {
			latestTs[pred.NodeID] = pred.Ts
		}
	}

	byNode := make(map[string]map[int32]DailyPrediction, len(latestTs))
	for _, pred := range predictions {
		if !pred.Ts.Equal(latestTs[pred.NodeID]) {
			continue
		}
		horizons := byNode[pred.NodeID]
		if horizons == nil {
			horizons = make(map[int32]DailyPrediction)
			byNode[pred.NodeID] = horizons
		}
		horizons[pred.HorizonDays] = pred
	}

	batches := make(map[string]dailyBatch, len(byNode))
	for nodeID, horizons := range byNode {
		items := make([]DailyPrediction, 0, len(horizons))
		for _, pred := range horizons {
			items = append(items, pred)
		}
		slices.SortFunc(items, func(a, b DailyPrediction) int {
			return cmp.Compare(a.HorizonDays, b.HorizonDays)
		})
		batches[nodeID] = dailyBatch{
			ts:          latestTs[nodeID],
			predictions: items,
		}
	}

	r.mu.Lock()
	if r.daily == nil {
		r.daily = make(map[string]dailyBatch)
	}
	for nodeID, batch := range batches {
		if existing, ok := r.daily[nodeID]; !ok || !batch.ts.Before(existing.ts) {
			r.daily[nodeID] = batch
		}
	}
	r.mu.Unlock()

	stored := int64(len(predictions))
	if r.log != nil {
		r.log.Debug("stored daily predictions", zap.Int64("stored", stored))
	}
	return stored, nil
}

// GetForecast returns the latest hourly and daily forecast predictions for a node, ordered by horizon.
func (r *Repository) GetForecast(_ context.Context, nodeID string) (Forecast, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var forecast Forecast
	if batch, ok := r.hourly[nodeID]; ok && len(batch.predictions) > 0 {
		forecast.Hourly = slices.Clone(batch.predictions)
	}
	if batch, ok := r.daily[nodeID]; ok && len(batch.predictions) > 0 {
		forecast.Daily = slices.Clone(batch.predictions)
	}
	return forecast, nil
}

// DeleteNode removes any stored hourly and daily predictions for nodeID from memory.
func (r *Repository) DeleteNode(nodeID string) {
	if nodeID == "" {
		return
	}
	r.mu.Lock()
	delete(r.hourly, nodeID)
	delete(r.daily, nodeID)
	r.mu.Unlock()
}

// DeleteNode evicts stored predictions for nodeID across all registered prediction repositories.
func DeleteNode(nodeID string) {
	if nodeID == "" {
		return
	}
	reposMu.RLock()
	for r := range repos {
		r.DeleteNode(nodeID)
	}
	reposMu.RUnlock()
}
