#include "Ingest.h"

#include <algorithm>
#include <cmath>
#include <cstdarg>
#include <cstdio>
#include <cstring>
#include <ctime>

namespace {

struct BufferWriter {
    char* data;
    size_t capacity;
    size_t written = 0;
    bool ok = true;

    bool append(const char* fmt, ...) {
        if (!ok || written >= capacity) {
            ok = false;
            return false;
        }
        va_list args;
        va_start(args, fmt);
        int const n = vsnprintf(data + written, capacity - written, fmt, args);
        va_end(args);
        if (n < 0 || static_cast<size_t>(n) >= capacity - written) {
            ok = false;
            return false;
        }
        written += static_cast<size_t>(n);
        data[written] = '\0';
        return true;
    }
};

// Append a raw field string with comma handling. Sets first false on success.
void appendRaw(BufferWriter& writer, bool& first, const char* fmt, ...) {
    va_list args;
    va_start(args, fmt);
    char field[64];
    int const n = vsnprintf(field, sizeof(field), fmt, args);
    va_end(args);
    if (n <= 0 || static_cast<size_t>(n) >= sizeof(field)) {
        writer.ok = false;
        return;
    }
    if (!first) {
        writer.append(",");
    }
    if (writer.ok) {
        writer.append("%s", field);
    }
    if (writer.ok) {
        first = false;
    }
}

void appendFloat(BufferWriter& writer, bool& first, const char* name, float value) {
    char number[24];
    if (formatFloat(number, sizeof(number), value) <= 0) {
        writer.ok = false;
        return;
    }
    appendRaw(writer, first, "\"%s\":%s", name, number);
}

void appendInt(BufferWriter& writer, bool& first, const char* name, long value) {
    appendRaw(writer, first, "\"%s\":%ld", name, value);
}

// Emit the magnet status group for one sensor.
void appendMagnetStatus(BufferWriter& writer, bool& first, const char* prefix,
                        const MagnetStatus& status) {
    if (status.status == weather::kInvalidMagnetStatus) {
        return;
    }
    char fieldName[24];
    snprintf(fieldName, sizeof(fieldName), "%s_status", prefix);
    appendInt(writer, first, fieldName, status.status);
    snprintf(fieldName, sizeof(fieldName), "%s_agc", prefix);
    appendInt(writer, first, fieldName, status.agc);
    snprintf(fieldName, sizeof(fieldName), "%s_agc_variance", prefix);
    appendInt(writer, first, fieldName, status.agcVariance);
}

// Append the full JSON object for one reading.
bool appendJsonReading(BufferWriter& writer, const IngestReading& reading, const char* nodeId) {
    if (!writer.append("{")) {
        return false;
    }
    bool first = true;
    appendRaw(writer, first, R"("node_id":"%s")", nodeId);

    if (reading.epoch != 0) {
        char timestamp[24];
        if (formatIso8601(reading.epoch, timestamp, sizeof(timestamp)) != 20) {
            return false;
        }
        appendRaw(writer, first, R"("timestamp":"%s")", timestamp);
    }

    const WindData& wind = reading.data.windData;
    if (wind.windSpeed != weather::kInvalidUint16) {
        appendFloat(writer, first, "wind_speed", weather::decodeWindSpeed(wind.windSpeed));
    }
    if (wind.windAngle != weather::kInvalidUint16) {
        appendFloat(writer, first, "wind_angle", weather::decodeWindAngle(wind.windAngle));
    }
    appendMagnetStatus(writer, first, "anemo", wind.anemometerMagnetStatus);
    appendMagnetStatus(writer, first, "vane", wind.windVaneMagnetStatus);

    const AirData& air = reading.data.airData;
    if (air.temperature != weather::kInvalidTemperature) {
        appendFloat(writer, first, "temperature", weather::decodeTemperature(air.temperature));
    }
    if (air.humidity != weather::kInvalidUint16) {
        appendFloat(writer, first, "humidity", weather::decodeHumidity(air.humidity));
    }
    if (air.pressure != weather::kInvalidUint16) {
        appendFloat(writer, first, "pressure", weather::decodePressure(air.pressure));
    }

    const RainData& rain = reading.data.rainData;
    if (rain.rainfall_1h != weather::kInvalidUint16) {
        appendFloat(writer, first, "rain_1h", weather::decodeRainAccumulation(rain.rainfall_1h));
    }
    if (rain.rainfall_24h != weather::kInvalidUint16) {
        appendFloat(writer, first, "rain_24h", weather::decodeRainAccumulation(rain.rainfall_24h));
    }
    if (rain.rainRate != weather::kInvalidUint16) {
        appendFloat(writer, first, "rain_rate", weather::decodeRainRate(rain.rainRate));
    }
    appendMagnetStatus(writer, first, "rain", rain.magnetStatus);

    const AirQualityData& airQuality = reading.data.airQualityData;
    if (airQuality.pm1_0 != weather::kInvalidUint16) {
        appendInt(writer, first, "pm1_0", airQuality.pm1_0);
    }
    if (airQuality.pm2_5 != weather::kInvalidUint16) {
        appendInt(writer, first, "pm2_5", airQuality.pm2_5);
    }
    if (airQuality.pm10_0 != weather::kInvalidUint16) {
        appendInt(writer, first, "pm10_0", airQuality.pm10_0);
    }
    if (airQuality.voc != weather::kInvalidUint16) {
        appendInt(writer, first, "voc", airQuality.voc);
    }

    return writer.append("}");
}

} // namespace

int formatFloat(char* out, size_t capacity, float value) {
    if (capacity == 0 || std::isnan(value) || std::isinf(value)) {
        return 0;
    }
    if (value == 0.0f) {
        value = 0.0f;
    }
    int n = snprintf(out, capacity, "%.2f", value);
    if (n <= 0 || static_cast<size_t>(n) >= capacity) {
        return 0;
    }
    char const* dot = strchr(out, '.');
    if (dot != nullptr) {
        char* end = out + n - 1;
        while (end > dot && *end == '0') {
            end--;
        }
        if (*end == '.') {
            *end = '\0';
            n = static_cast<int>(end - out);
            return n;
        }
        end++;
        *end = '\0';
        n = static_cast<int>(end - out);
    }
    return n;
}

int formatIso8601(uint32_t epoch, char* out, size_t capacity) {
    if (capacity < 21) {
        return 0;
    }
    auto const seconds = static_cast<time_t>(epoch);
    struct tm utc;
    if (gmtime_r(&seconds, &utc) == nullptr) {
        return 0;
    }
    int const n = snprintf(out, capacity, "%04d-%02d-%02dT%02d:%02d:%02dZ", utc.tm_year + 1900,
                           utc.tm_mon + 1, utc.tm_mday, utc.tm_hour, utc.tm_min, utc.tm_sec);
    if (n != 20) {
        return 0;
    }
    return n;
}

bool sanitizeNodeId(const char* raw, char* out, size_t capacity) {
    if (raw == nullptr || out == nullptr || capacity == 0) {
        return false;
    }
    size_t written = 0;
    for (const char* p = raw; *p != '\0' && written + 1 < capacity; ++p) {
        char const c = *p;
        bool const alnum =
            (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9');
        if (alnum || c == '-' || c == '_') {
            out[written++] = c;
        }
    }
    if (written == 0) {
        return false;
    }
    out[written] = '\0';
    return true;
}

int buildSinglePayload(const IngestReading& reading, const char* nodeId, char* out,
                       size_t capacity) {
    if (out == nullptr || capacity == 0) {
        return 0;
    }
    BufferWriter writer{out, capacity};
    if (!appendJsonReading(writer, reading, nodeId)) {
        return 0;
    }
    return static_cast<int>(writer.written);
}

int buildBatchPayload(const IngestReading* readings, size_t count, const char* nodeId, char* out,
                      size_t capacity) {
    if (readings == nullptr || count == 0 || out == nullptr || capacity == 0) {
        return 0;
    }
    BufferWriter writer{out, capacity};
    if (!writer.append("[")) {
        return 0;
    }
    for (size_t i = 0; i < count; ++i) {
        if (i > 0 && !writer.append(",")) {
            return 0;
        }
        if (!appendJsonReading(writer, readings[i], nodeId)) {
            return 0;
        }
    }
    if (!writer.append("]")) {
        return 0;
    }
    return static_cast<int>(writer.written);
}

uint32_t backoffDelayMs(const BackoffPolicy& policy, uint8_t retryIndex) {
    if (retryIndex == 0) {
        return 0;
    }
    uint64_t delay = policy.minDelayMs;
    for (uint8_t i = 1; i < retryIndex && delay < policy.maxDelayMs; ++i) {
        delay *= policy.multiplier;
    }
    delay = std::min<uint64_t>(delay, policy.maxDelayMs);
    return static_cast<uint32_t>(delay);
}

bool shouldRetry(const BackoffPolicy& policy, uint8_t retryIndex) {
    return policy.maxAttempts == 0 || retryIndex < policy.maxAttempts;
}
