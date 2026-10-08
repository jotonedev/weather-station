#ifndef SOFT_TWO_WIRE_H
#define SOFT_TWO_WIRE_H

#include <Arduino.h>
#include <Wire.h>
#include <SoftWire.h>

/**
 * @brief TwoWire adapter for SoftWire (software / bit-banged I2C).
 * Allows any library expecting a pointer or reference to Arduino TwoWire (such as AS5600)
 * to run transparently over bit-banged software I2C on arbitrary GPIO pins.
 */
class SoftTwoWire : public TwoWire {
public:
    SoftTwoWire(uint8_t sdaPin, uint8_t sclPin)
        : TwoWire(255), _softWire(sdaPin, sclPin) {
        _softWire.setRxBuffer(_rxBuffer, sizeof(_rxBuffer));
        _softWire.setTxBuffer(_txBuffer, sizeof(_txBuffer));
    }

    bool beginSoftwareI2C() {
        _softWire.begin();
        return true;
    }

    bool beginSoftwareI2C(int sda, int scl, uint32_t frequency = 0) {
        if (sda >= 0) _softWire.setSda(sda);
        if (scl >= 0) _softWire.setScl(scl);
        if (frequency > 0) _softWire.setClock(frequency);
        _softWire.begin();
        return true;
    }

    bool end() override {
        _softWire.end();
        return true;
    }

    bool setClock(uint32_t freq) override {
        _softWire.setClock(freq);
        return true;
    }

    void beginTransmission(uint8_t address) override {
        _softWire.beginTransmission(address);
    }

    uint8_t endTransmission(bool stopBit) override {
        return _softWire.endTransmission(stopBit);
    }

    uint8_t endTransmission(void) override {
        return _softWire.endTransmission(true);
    }

    size_t requestFrom(uint8_t address, size_t len, bool stopBit) override {
        return _softWire.requestFrom(address, (uint8_t)len, (uint8_t)stopBit);
    }

    size_t requestFrom(uint8_t address, size_t len) override {
        return _softWire.requestFrom(address, (uint8_t)len, (uint8_t)true);
    }

    size_t write(uint8_t data) override {
        return _softWire.write(data);
    }

    size_t write(const uint8_t *data, size_t quantity) override {
        return _softWire.write(data, quantity);
    }

    int available(void) override {
        return _softWire.available();
    }

    int read(void) override {
        return _softWire.read();
    }

    int peek(void) override {
        return _softWire.peek();
    }

    void flush(void) override {
        _softWire.flush();
    }

    void onReceive(const std::function<void(int)> &) override {}
    void onRequest(const std::function<void()> &) override {}

    SoftWire& getSoftWire() { return _softWire; }

private:
    SoftWire _softWire;
    uint8_t _rxBuffer[32];
    uint8_t _txBuffer[32];
};

#endif // SOFT_TWO_WIRE_H
