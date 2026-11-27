#ifndef SOFT_TWO_WIRE_H
#define SOFT_TWO_WIRE_H

#include <Arduino.h>
#include <SoftWire.h>
#include <Wire.h>

#include "constants.h"

inline bool recoverI2cBusPins(uint8_t sdaPin, uint8_t sclPin, bool enablePullups = true) {
    const uint8_t inputMode = enablePullups ? INPUT_PULLUP : INPUT;
    pinMode(sdaPin, inputMode);
    pinMode(sclPin, inputMode);
    delayMicroseconds(5);

    if (digitalRead(sdaPin) == HIGH && digitalRead(sclPin) == HIGH) {
        return true;
    }

    for (int i = 0; i < 9; ++i) {
        digitalWrite(sclPin, LOW);
        pinMode(sclPin, OUTPUT);
        delayMicroseconds(5);
        pinMode(sclPin, inputMode);
        delayMicroseconds(5);
        if (digitalRead(sdaPin) == HIGH && digitalRead(sclPin) == HIGH) {
            break;
        }
    }

    // Send a clean STOP condition: SCL LOW -> SDA LOW -> SCL release -> SDA release.
    digitalWrite(sclPin, LOW);
    pinMode(sclPin, OUTPUT);
    delayMicroseconds(5);
    digitalWrite(sdaPin, LOW);
    pinMode(sdaPin, OUTPUT);
    delayMicroseconds(5);
    pinMode(sclPin, inputMode);
    delayMicroseconds(5);
    pinMode(sdaPin, inputMode);
    delayMicroseconds(5);

    return digitalRead(sdaPin) == HIGH && digitalRead(sclPin) == HIGH;
}

/**
 * @brief TwoWire adapter for SoftWire (software / bit-banged I2C).
 * Allows any library expecting a pointer or reference to Arduino TwoWire (such as AS5600)
 * to run transparently over bit-banged software I2C on arbitrary GPIO pins.
 */
class SoftTwoWire : public TwoWire {
  public:
    SoftTwoWire(uint8_t sdaPin, uint8_t sclPin) : TwoWire(255), _softWire(sdaPin, sclPin) {
        _softWire.setRxBuffer(_rxBuffer, sizeof(_rxBuffer));
        _softWire.setTxBuffer(_txBuffer, sizeof(_txBuffer));
    }

    bool beginSoftwareI2C() {
        _softWire.enablePullups(true);
        _softWire.setTimeout_ms(I2C_TIMEOUT_MS);
        _softWire.setClock(I2C_CLOCK_HZ);
        const bool idle = recoverBus();
        _softWire.begin();
        return idle;
    }

    bool beginSoftwareI2C(int sda, int scl, uint32_t frequency = 0) {
        if (sda >= 0) {
            _softWire.setSda(sda);
        }
        if (scl >= 0) {
            _softWire.setScl(scl);
        }
        _softWire.enablePullups(true);
        _softWire.setTimeout_ms(I2C_TIMEOUT_MS);
        _softWire.setClock(frequency > 0 ? frequency : I2C_CLOCK_HZ);
        const bool idle = recoverBus();
        _softWire.begin();
        return idle;
    }

    bool recoverBus() {
        const bool idle = recoverI2cBusPins(_softWire.getSda(), _softWire.getScl(), true);
        _inTransaction = false;
        return idle;
    }

    bool end() override {
        _inTransaction = false;
        _softWire.end();
        return true;
    }

    bool setClock(uint32_t freq) override {
        _softWire.setClock(freq);
        return true;
    }

    void beginTransmission(uint8_t address) override { _softWire.beginTransmission(address); }

    uint8_t endTransmission(bool stopBit) override {
        if (!ensureBusIdle()) {
            return 4;
        }
        const uint8_t res = _softWire.endTransmission(stopBit);
        _inTransaction = !stopBit;
        return res;
    }

    uint8_t endTransmission(void) override { return endTransmission(true); }

    size_t requestFrom(uint8_t address, size_t len, bool stopBit) override {
        if (!ensureBusIdle()) {
            return 0;
        }
        const size_t res = _softWire.requestFrom(address, static_cast<uint8_t>(len),
                                                 static_cast<uint8_t>(stopBit));
        _inTransaction = !stopBit;
        return res;
    }

    size_t requestFrom(uint8_t address, size_t len) override {
        return requestFrom(address, len, true);
    }

    size_t write(uint8_t data) override { return _softWire.write(data); }

    size_t write(const uint8_t* data, size_t quantity) override {
        return _softWire.write(data, quantity);
    }

    int available(void) override { return _softWire.available(); }

    int read(void) override { return _softWire.read(); }

    int peek(void) override { return _softWire.peek(); }

    void flush(void) override { _softWire.flush(); }

    void onReceive(const std::function<void(int)>&) override {}
    void onRequest(const std::function<void()>&) override {}

    SoftWire& getSoftWire() { return _softWire; }

  private:
    bool ensureBusIdle() {
        if (_inTransaction) {
            return true;
        }
        if (digitalRead(_softWire.getSda()) == HIGH && digitalRead(_softWire.getScl()) == HIGH) {
            return true;
        }
        return recoverBus();
    }

    SoftWire _softWire;
    uint8_t _rxBuffer[32];
    uint8_t _txBuffer[32];
    bool _inTransaction = false;
};

#endif // SOFT_TWO_WIRE_H
