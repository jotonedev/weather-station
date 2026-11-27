/*!
 *  @file Adafruit_AHTX0.h
 *
 * 	I2C Driver for the Adafruit AHT10 / AHT20 Humidity and Temperature
 *Sensor library
 *
 * 	This is a library for the Adafruit AHT20 breakout:
 * 	https://www.adafruit.com/products/4566
 *
 * 	Adafruit invests time and resources providing this open source code,
 *  please support Adafruit and open-source hardware by purchasing products from
 * 	Adafruit!
 *
 *
 *	BSD license (see license.txt)
 */

#ifndef _ADAFRUIT_AHTX0_H
#define _ADAFRUIT_AHTX0_H

#ifdef ARDUINO

#include "Aht20Codec.h"
#include "Arduino.h"
#include <Sync.h>
#include <Wire.h>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>

#define AHTX0_I2CADDR_DEFAULT 0x38   ///< AHT default i2c address
#define AHTX0_I2CADDR_ALTERNATE 0x39 ///< AHT alternate i2c address
#define AHTX0_CMD_CALIBRATE 0xBE     ///< AHT20 calibration command
#define AHTX0_CMD_TRIGGER 0xAC       ///< Trigger reading command
#define AHTX0_CMD_SOFTRESET 0xBA     ///< Soft reset command
#define AHTX0_STATUS_BUSY 0x80       ///< Status bit for busy
#define AHTX0_STATUS_CALIBRATED 0x08 ///< Status bit for calibrated

/*!
 *    @brief  Class that stores state and functions for interacting with
 *            the AHT10/AHT20 I2C Temperature/Humidity sensor
 */
class Adafruit_AHTX0 {
public:
  Adafruit_AHTX0();
  ~Adafruit_AHTX0();

  bool begin(TwoWire *wire = &Wire, SemaphoreHandle_t busMutex = nullptr,
             uint8_t i2c_address = AHTX0_I2CADDR_DEFAULT);
  void setBusMutex(SemaphoreHandle_t busMutex);
  bool isOk() const;

  bool read(aht20::Aht20Reading &out);
  uint8_t getStatus(void);

private:
  bool writeBytes(const uint8_t *data, size_t len);
  bool readBytes(uint8_t *data, size_t len);
  uint8_t getStatusLocked(void);
  bool waitNotBusy(uint32_t maxAttempts, uint32_t delayMs);

  TwoWire *_wire = nullptr;
  SemaphoreHandle_t _busMutex = nullptr;
  uint8_t _addr = AHTX0_I2CADDR_DEFAULT;
  bool _ok = false;
};

#endif // ARDUINO

#endif
