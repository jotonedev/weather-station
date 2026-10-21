#ifndef CONSTANTS_H
#define CONSTANTS_H

// =============================================================================
// ESP32-S3 Clean & Conflict-Free Pin Assignment
// Avoids:
//   - Flash / PSRAM bus: GPIO 26–37 (strictly reserved on ESP32-S3)
//   - Native USB (D- / D+): GPIO 19, 20
//   - Strapping pins: GPIO 0, 3, 45, 46
//   - Hardware UART0 console: GPIO 43, 44
//   - Onboard RGB LED: GPIO 38 / 48
// =============================================================================

// --- CC1101 Radio SPI ---
#define RADIO_CS_PIN 10
#define MOSI_PIN 11
#define SCK_PIN 12
#define MISO_PIN 13
#define RADIO_GDO0_PIN 14

// --- I2C Bus 1 (Hardware Wire) ---
// Shared: Anemometer AS5600 (@ 0x36) + Environmental: AHT20 (@ 0x38), SGP30 (@ 0x58), BMP180 (@
// 0x77)
#define I2C_1_SDA_PIN 4
#define I2C_1_SCL_PIN 5

#define SENSOR_I2C_SDA_PIN I2C_1_SDA_PIN
#define SENSOR_I2C_SCL_PIN I2C_1_SCL_PIN

#define ANEMOMETER_DIRECTION_PIN 6

// --- I2C Bus 2 (Hardware Wire1) ---
// Dedicated: Rain Meter AS5600 (@ 0x36)
#define RAIN_METER_SDA_PIN 8
#define RAIN_METER_SCL_PIN 9
#define RAIN_METER_DIRECTION_PIN 1

// Funnel radius r = 55 mm -> Area A = π * r² = 9503.3178 mm²
// Bucket volume V = 18 ml = 18,000 mm³
// Rainfall depth per tip = V / A = 18,000 / (π * 55²) ≈ 1.894075 mm
#define RAIN_METER_FUNNEL_RADIUS_MM 55.0f
#define RAIN_METER_BUCKET_VOLUME_ML 18.0f
#define RAIN_METER_CALIBRATION_MM_PER_TIP 1.894075f

// --- I2C Bus 3 (Software I2C / SoftWire) ---
// Dedicated: Wind Vane AS5600 (@ 0x36)
#define WIND_VANE_SDA_PIN 15
#define WIND_VANE_SCL_PIN 16
#define WIND_VANE_DIRECTION_PIN 7
#define WIND_VANE_CALIBRATION_OFFSET 0.0f

// --- Calibration settings ---
// Set to 1 to disable radio transmissions and stream live sensor telemetry to Serial.
// Set to 0 for normal production operation.
#define CALIBRATION_MODE 1

// Speed factor for anemometer: m/s per RPM (calculated from cup geometry R=100mm, cup factor f=3.0)
#define ANEMOMETER_SPEED_FACTOR 0.032f

// --- PMS7003 Particulate Matter Sensor (UART2) ---
#define PMS_RX_PIN 17 // Connects to PMS7003 TX
#define PMS_TX_PIN 18 // Connects to PMS7003 RX

// =============================================================================
// Reporting and link policy
// =============================================================================

// How often a reading is sampled and scheduled for the radio link.
#define REPORT_INTERVAL_MS 60000uL
// How long a send waits for an ACK before ARQ retransmits.
#define TX_ACK_TIMEOUT_MS 1000uL
// Attempts per reading before it is dropped.
#define TX_MAX_ATTEMPTS 3
// Delay before retrying a busy or failed send attempt.
#define TX_BACKOFF_MS 2000uL
// Upper bound for one attempt to settle, forces a radio re-sync when exceeded.
#define TX_HARD_TIMEOUT_MS 12000uL
// Consecutive dropped readings before the radio is initialized again.
#define RADIO_REINIT_THRESHOLD 3
// The board restarts when the loop stalls this long.
#define WATCHDOG_TIMEOUT_MS 10000uL
// Hardware and software I2C transaction timeout to recover from bus glitches.
#define I2C_TIMEOUT_MS 20u
#define I2C_CLOCK_HZ 100000uL

// PMS7003 duty-cycling policy: wake for 30s every 15 minutes to preserve laser and fan lifespan.
#define PMS_SAMPLE_INTERVAL_MS 900000uL
#define PMS_WARMUP_MS 30000uL
#define PMS_READ_TIMEOUT_MS 10000uL

// SGP30 requires a 1 Hz IAQmeasure tick and periodic baseline persistence to NVS.
#define SGP_MEASURE_INTERVAL_MS 1000uL
#define SGP_WARMUP_BEFORE_SAVE_MS 3600000uL
#define SGP_BASELINE_SAVE_INTERVAL_MS 21600000uL

// NVS keys for the boot counter, last failure tag, and SGP30 IAQ baseline.
#define NVS_NAMESPACE "control"
#define NVS_KEY_BOOTS "boots"
#define NVS_KEY_LAST_ERR "lasterr"
#define NVS_KEY_SGP_ECO2 "sgp_eco2"
#define NVS_KEY_SGP_TVOC "sgp_tvoc"
// Boots in a row without a successful send flag a crash loop.
#define BOOT_CRASH_THRESHOLD 4

#endif // CONSTANTS_H