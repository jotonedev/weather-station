#ifndef PMS_H
#define PMS_H

#ifdef ARDUINO

#include "PmsDutyCycle.h"
#include "Stream.h"

class PMS
{
public:
  static const uint16_t SINGLE_RESPONSE_TIME = 1000;
  static const uint16_t TOTAL_RESPONSE_TIME = 1000 * 10;
  static const uint16_t STEADY_RESPONSE_TIME = 1000 * 30;

  static const uint16_t BAUD_RATE = 9600;

  struct DATA {
    // Standard Particles, CF=1
    uint16_t PM_SP_UG_1_0;
    uint16_t PM_SP_UG_2_5;
    uint16_t PM_SP_UG_10_0;

    // Atmospheric environment
    uint16_t PM_AE_UG_1_0;
    uint16_t PM_AE_UG_2_5;
    uint16_t PM_AE_UG_10_0;
  };

  PMS(Stream&);
  void sleep();
  void wakeUp();
  void activeMode();
  void passiveMode();

  void requestRead();
  void reset();
  bool read(DATA& data);
  bool readUntil(DATA& data, uint16_t timeout = SINGLE_RESPONSE_TIME);

  uint8_t lastErrorCode() const;
  uint32_t sensorErrorCount() const;
  uint32_t checksumErrorCount() const;
  uint32_t ackCount() const;

private:
  enum MODE { MODE_ACTIVE, MODE_PASSIVE };

  Stream* _stream;
  MODE _mode = MODE_ACTIVE;
  pms7003::PmsStreamParser _parser;
};

class Pms7003Driver
{
public:
  explicit Pms7003Driver(Stream& stream, const pms7003::PmsConfig& config = pms7003::PmsConfig{});

  void begin(uint32_t nowMs);
  pms7003::PmsActions poll(uint32_t nowMs);
  bool hasPendingReport(uint32_t nowMs);
  const pms7003::PmsFrame& latestFrame() const;
  void markReportDelivered();
  pms7003::PmsState state() const;
  bool timedOut() const;
  bool hadError() const;

private:
  void sendCommand(const uint8_t cmd[7]);
  void flushRx();

  Stream& _stream;
  pms7003::PmsDutyCycle _dutyCycle;
};

#endif // ARDUINO

#endif