/*!
 *  @file Adafruit_AHTX0.cpp
 *
 *  @mainpage Adafruit AHTX0 Humidity and Temperature Sensor library
 *
 *  @section intro_sec Introduction
 *
 * 	I2C Driver for the Adafruit AHTX0 Humidity and Temperature Sensor
 * library
 *
 * 	This is a library for the Adafruit AHT20 breakout:
 * 	https://www.adafruit.com/product/4566
 *
 * 	Adafruit invests time and resources providing this open source code,
 *  please support Adafruit and open-source hardware by purchasing products from
 * 	Adafruit!
 *
 *  @section dependencies Dependencies
 *  This library depends on the Adafruit BusIO library
 *
 *  This library depends on the Adafruit Unified Sensor library
 *
 *  @section author Author
 *
 *  Limor Fried (Adafruit Industries)
 *
 * 	@section license License
 *
 * 	BSD (see license.txt)
 *
 * 	@section  HISTORY
 *
 *     v1.0 - First release
 */

#ifdef ARDUINO

#include "Arduino.h"

#include "Adafruit_AHTX0.h"

/*!
 *    @brief  Instantiates a new AHTX0 class
 */
Adafruit_AHTX0::Adafruit_AHTX0(void) {}

Adafruit_AHTX0::~Adafruit_AHTX0(void) {}

void Adafruit_AHTX0::setBusMutex(SemaphoreHandle_t busMutex) {
  _busMutex = busMutex;
}

bool Adafruit_AHTX0::isOk() const { return _ok; }

bool Adafruit_AHTX0::writeBytes(const uint8_t *data, size_t len) {
  if (_wire == nullptr || data == nullptr || len == 0) {
    return false;
  }
  _wire->beginTransmission(_addr);
  if (_wire->write(data, len) != len) {
    _wire->endTransmission(true);
    return false;
  }
  return _wire->endTransmission(true) == 0;
}

bool Adafruit_AHTX0::readBytes(uint8_t *data, size_t len) {
  if (_wire == nullptr || data == nullptr || len == 0) {
    return false;
  }
  const size_t received = _wire->requestFrom(_addr, len);
  if (received != len) {
    while (_wire->available() > 0) {
      (void)_wire->read();
    }
    return false;
  }
  for (size_t i = 0; i < len; ++i) {
    const int b = _wire->read();
    if (b < 0) {
      return false;
    }
    data[i] = static_cast<uint8_t>(b);
  }
  return true;
}

uint8_t Adafruit_AHTX0::getStatusLocked(void) {
  uint8_t ret = 0xFF;
  if (!readBytes(&ret, 1)) {
    return 0xFF;
  }
  return ret;
}

bool Adafruit_AHTX0::waitNotBusy(uint32_t maxAttempts, uint32_t delayMs) {
  for (uint32_t attempt = 0; attempt < maxAttempts; ++attempt) {
    uint8_t status = 0xFF;
    {
      MutexGuard guard(_busMutex);
      status = getStatusLocked();
    }
    if ((status & AHTX0_STATUS_BUSY) == 0u) {
      return true;
    }
    vTaskDelay(pdMS_TO_TICKS(delayMs));
  }
  return false;
}

/*!
 *    @brief  Sets up the hardware and initializes I2C
 *    @param  wire
 *            The Wire object to be used for I2C connections.
 *    @param  busMutex
 *            Optional FreeRTOS mutex protecting the shared I2C bus.
 *    @param  i2c_address
 *            The I2C address used to communicate with the sensor
 *    @return True if initialization was successful, otherwise false.
 */
bool Adafruit_AHTX0::begin(TwoWire *wire, SemaphoreHandle_t busMutex,
                           uint8_t i2c_address) {
  _wire = wire;
  _busMutex = busMutex;
  _addr = i2c_address;
  _ok = false;

  if (_wire == nullptr) {
    return false;
  }

  vTaskDelay(pdMS_TO_TICKS(40)); // 40 ms to power up

  uint8_t cmd[3];
  cmd[0] = AHTX0_CMD_SOFTRESET;
  {
    MutexGuard guard(_busMutex);
    if (!writeBytes(cmd, 1)) {
      return false;
    }
  }
  vTaskDelay(pdMS_TO_TICKS(20));

  if (!waitNotBusy(10, 10)) {
    return false;
  }

  uint8_t status = 0xFF;
  {
    MutexGuard guard(_busMutex);
    status = getStatusLocked();
  }
  if ((status & AHTX0_STATUS_CALIBRATED) == 0u) {
    cmd[0] = AHTX0_CMD_CALIBRATE;
    cmd[1] = 0x08;
    cmd[2] = 0x00;
    {
      MutexGuard guard(_busMutex);
      (void)writeBytes(cmd, 3); // may not 'succeed' on newer AHT20s
    }
    if (!waitNotBusy(10, 10)) {
      return false;
    }
    {
      MutexGuard guard(_busMutex);
      status = getStatusLocked();
    }
    if ((status & AHTX0_STATUS_CALIBRATED) == 0u) {
      return false;
    }
  }

  _ok = true;
  return true;
}

/**
 * @brief  Gets the status (first byte) from AHT10/AHT20
 *
 * @returns 8 bits of status data, or 0xFF if failed
 */
uint8_t Adafruit_AHTX0::getStatus(void) {
  MutexGuard guard(_busMutex);
  return getStatusLocked();
}

bool Adafruit_AHTX0::read(aht20::Aht20Reading &out) {
  if (!_ok || _wire == nullptr) {
    return false;
  }

  const uint8_t cmd[3] = {AHTX0_CMD_TRIGGER, 0x33, 0x00};
  {
    MutexGuard guard(_busMutex);
    if (!writeBytes(cmd, sizeof(cmd))) {
      return false;
    }
  }

  // Wait outside busMutex for the 80 ms conversion.
  vTaskDelay(pdMS_TO_TICKS(80));

  for (uint32_t attempt = 0; attempt < 5; ++attempt) {
    uint8_t frame[7] = {};
    {
      MutexGuard guard(_busMutex);
      if (!readBytes(frame, sizeof(frame))) {
        return false;
      }
    }
    if ((frame[0] & AHTX0_STATUS_BUSY) == 0u) {
      return aht20::decodeAht20Frame(frame, out);
    }
    vTaskDelay(pdMS_TO_TICKS(10));
  }
  return false;
}

#endif // ARDUINO
