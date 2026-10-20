#ifndef BASE_CONFIG_H
#define BASE_CONFIG_H

// =============================================================================
// Networking
// =============================================================================

// WiFi credentials. Set these before flashing the base station.
#define WIFI_SSID "your-ssid"
#define WIFI_PASSWORD "your-password"

// Ingest endpoint. Use http for a LAN deployment, https for the field. The
// https client trusts the Arduino core certificate bundle.
#define INGEST_URL "http://192.168.1.50:8081/readings"
// #define INGEST_URL "https://ingest.example.com/readings"

// Node id reported to the ingest service.
#define NODE_ID "weather-mast"

// NTP server used to timestamp received readings.
#define NTP_SERVER "pool.ntp.org"
#define NTP_TIMEOUT_MS 5000uL
#define WIFI_CONNECT_TIMEOUT_MS 15000uL
#define WIFI_RECONNECT_DELAY_MS 5000uL

// =============================================================================
// Radio pins (LOLIN S3 Mini)
// =============================================================================

#define RADIO_CS_PIN   16
#define RADIO_GDO0_PIN 18
#define SCK_PIN        12
#define MISO_PIN       13
#define MOSI_PIN       11

constexpr uint8_t CONTROL_NODE_ADDR = 1;
constexpr uint8_t BASE_STATION_ADDR = 2;

// =============================================================================
// Delivery policy
// =============================================================================

// How many readings the outbox holds while the network is unavailable. At one
// reading per minute this is roughly two hours of buffering.
#define OUTBOX_CAPACITY 120
// Largest batch flushed in one HTTP request.
#define BATCH_FLUSH_MAX 12
// Payload buffer for a full batch. Keep above the largest possible payload.
#define INGEST_BUFFER_SIZE 4096
// HTTP connect and transfer timeout.
#define HTTP_TIMEOUT_MS 8000uL
// Delay between outbox polls when idle.
#define FLUSH_IDLE_DELAY_MS 2000uL
// Exponential backoff bounds for transient failures.
#define BACKOFF_MIN_MS 2000uL
#define BACKOFF_MAX_MS 120000uL
#define BACKOFF_MULTIPLIER 3

// =============================================================================
// NVS counters
// =============================================================================

#define NVS_NAMESPACE "base"
#define NVS_KEY_RECEIVED "recv"
#define NVS_KEY_SENT "sent"
#define NVS_KEY_DROPPED "drop"
#define NVS_KEY_FRAMES_INVALID "binv"
#define NVS_KEY_BUFFER_DROPPED "bdrop"
#define NVS_KEY_LAST_HTTP "http"

#endif  // BASE_CONFIG_H
