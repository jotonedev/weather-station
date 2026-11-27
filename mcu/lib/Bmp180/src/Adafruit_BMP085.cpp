/*!
 * @file Adafruit_BMP085.cpp
 *
 * @mainpage Adafruit BMP085 Library
 *
 * @section intro_sec Introduction
 *
 * This is a library for the Adafruit BMP085/BMP180 Barometric Pressure + Temp
 * sensor
 *
 * Designed specifically to work with the Adafruit BMP085 or BMP180 Breakout
 * ----> http://www.adafruit.com/products/391
 * ----> http://www.adafruit.com/products/1603
 *
 * These displays use I2C to communicate, 2 pins are required to
 * interface
 * Adafruit invests time and resources providing this open source code,
 * please support Adafruit and open-source hardware by purchasing
 * products from Adafruit!
 *
 * @section author Author
 *
 * Written by Limor Fried/Ladyada for Adafruit Industries.
 * Updated by Samy Kamkar for cross-platform support.
 *
 * @section license License
 *
 * BSD license, all text above must be included in any redistribution
 */

#ifdef ARDUINO

#include "Adafruit_BMP085.h"

Adafruit_BMP085::Adafruit_BMP085() {}

void Adafruit_BMP085::setBusMutex(SemaphoreHandle_t busMutex) {
  _busMutex = busMutex;
}

bool Adafruit_BMP085::isOk() const { return _ok; }

bool Adafruit_BMP085::writeRegister(uint8_t reg, uint8_t value) {
  if (_wire == nullptr) {
    return false;
  }
  _wire->beginTransmission(BMP085_I2CADDR);
  _wire->write(reg);
  _wire->write(value);
  return _wire->endTransmission(true) == 0;
}

bool Adafruit_BMP085::readBytes(uint8_t reg, uint8_t *dst, size_t len) {
  if (_wire == nullptr || dst == nullptr || len == 0) {
    return false;
  }
  _wire->beginTransmission(BMP085_I2CADDR);
  _wire->write(reg);
  if (_wire->endTransmission(false) != 0) {
    return false;
  }
  const size_t received = _wire->requestFrom(BMP085_I2CADDR, len);
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
    dst[i] = static_cast<uint8_t>(b);
  }
  return true;
}

bool Adafruit_BMP085::begin(uint8_t mode, TwoWire *wire,
                            SemaphoreHandle_t busMutex) {
  if (mode > BMP085_ULTRAHIGHRES) {
    mode = BMP085_ULTRAHIGHRES;
  }
  oversampling = mode;
  _wire = wire;
  _busMutex = busMutex;
  _ok = false;

  if (_wire == nullptr) {
    return false;
  }

  uint8_t chipId = 0;
  uint8_t rawCal[22] = {};
  {
    MutexGuard guard(_busMutex);
    if (!readBytes(BMP085_CHIPID_REG, &chipId, 1) ||
        chipId != BMP085_CHIPID_VAL) {
      return false;
    }
    if (!readBytes(BMP085_CAL_AC1, rawCal, sizeof(rawCal))) {
      return false;
    }
  }

  if (!bmp180::parseBmp180Calibration(rawCal, _cal)) {
    return false;
  }

  _ok = true;
  return true;
}

bool Adafruit_BMP085::readRawTemperature(int32_t &utOut) {
  {
    MutexGuard guard(_busMutex);
    if (!writeRegister(BMP085_CONTROL, BMP085_READTEMPCMD)) {
      return false;
    }
  }
  // Wait outside busMutex for 5 ms UT conversion.
  vTaskDelay(pdMS_TO_TICKS(5));

  uint8_t buf[2] = {};
  {
    MutexGuard guard(_busMutex);
    if (!readBytes(BMP085_TEMPDATA, buf, sizeof(buf))) {
      return false;
    }
  }
  utOut = (static_cast<int32_t>(buf[0]) << 8) | static_cast<int32_t>(buf[1]);
  return true;
}

bool Adafruit_BMP085::readRawPressure(int32_t &upOut) {
  {
    MutexGuard guard(_busMutex);
    if (!writeRegister(BMP085_CONTROL,
                       static_cast<uint8_t>(BMP085_READPRESSURECMD +
                                            (oversampling << 6)))) {
      return false;
    }
  }

  // Wait outside busMutex for UP conversion.
  uint32_t waitMs = 26;
  if (oversampling == BMP085_ULTRALOWPOWER) {
    waitMs = 5;
  } else if (oversampling == BMP085_STANDARD) {
    waitMs = 8;
  } else if (oversampling == BMP085_HIGHRES) {
    waitMs = 14;
  }
  vTaskDelay(pdMS_TO_TICKS(waitMs));

  uint8_t buf[3] = {};
  {
    MutexGuard guard(_busMutex);
    if (!readBytes(BMP085_PRESSUREDATA, buf, sizeof(buf))) {
      return false;
    }
  }
  const uint32_t raw = (static_cast<uint32_t>(buf[0]) << 16) |
                       (static_cast<uint32_t>(buf[1]) << 8) |
                       static_cast<uint32_t>(buf[2]);
  upOut = static_cast<int32_t>(raw >> (8 - oversampling));
  return true;
}

bool Adafruit_BMP085::readPressurePa(int32_t &pressurePaOut) {
  if (!_ok || _wire == nullptr) {
    return false;
  }

  int32_t ut = 0;
  int32_t up = 0;
  if (!readRawTemperature(ut) || !readRawPressure(up)) {
    return false;
  }
  return bmp180::computeBmp180PressurePa(_cal, ut, up, oversampling,
                                         pressurePaOut);
}

#endif // ARDUINO
