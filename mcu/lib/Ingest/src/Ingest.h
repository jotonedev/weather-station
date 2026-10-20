#ifndef INGEST_H
#define INGEST_H

#include <cstddef>
#include <cstdint>

#include "Weather.h"

// IngestReading carries the data and timestamp for one JSON payload.
struct IngestReading {
    WeatherData data;
    // Unix epoch seconds. Zero omits the timestamp so the ingest service
    // stamps the reading with its own clock.
    uint32_t epoch;
};

// BackoffPolicy drives the retry schedule for failed deliveries.
struct BackoffPolicy {
    uint32_t minDelayMs = 1000;
    uint32_t maxDelayMs = 60000;
    uint32_t multiplier = 2;
    // Zero means retry forever.
    uint8_t maxAttempts = 0;
};

// Write value with up to 2 decimals and no trailing zeros. Returns the bytes
// written or 0 when value is not representable (NaN, infinite, small buffer).
int formatFloat(char* out, size_t capacity, float value);

// Write an RFC 3339 UTC timestamp (YYYY-MM-DDTHH:MM:SSZ). Returns 20 on
// success or 0 when the buffer is too small.
int formatIso8601(uint32_t epoch, char* out, size_t capacity);

// Copy raw into out keeping only alphanumerics, dash and underscore. Returns
// false when nothing survives the filter or the buffer is too small.
bool sanitizeNodeId(const char* raw, char* out, size_t capacity);

// Build a single reading JSON payload. Returns the bytes written or 0 on
// failure. nodeId must already be sanitized.
int buildSinglePayload(const IngestReading& reading, const char* nodeId,
                       char* out, size_t capacity);

// Build a JSON array payload for a batch of readings. Returns the bytes
// written or 0 on failure or an empty batch.
int buildBatchPayload(const IngestReading* readings, size_t count, const char* nodeId,
                      char* out, size_t capacity);

// Delay to wait before retry at the given retry index (0 is the first, wait
// nothing).
uint32_t backoffDelayMs(const BackoffPolicy& policy, uint8_t retryIndex);

// Whether another attempt is allowed after retryIndex failures.
bool shouldRetry(const BackoffPolicy& policy, uint8_t retryIndex);

#endif  // INGEST_H
