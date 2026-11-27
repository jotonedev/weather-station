#ifdef ARDUINO

#include "Arduino.h"
#include "PMS.h"

PMS::PMS(Stream& stream)
{
  this->_stream = &stream;
}

// Standby mode. For low power consumption and prolong the life of the sensor.
void PMS::sleep()
{
  _stream->write(pms7003::kCmdSleep, sizeof(pms7003::kCmdSleep));
  _stream->flush();
}

// Operating mode. Stable data should be got at least 30 seconds after the sensor wakeup from the sleep mode because of the fan's performance.
void PMS::wakeUp()
{
  _stream->write(pms7003::kCmdWakeUp, sizeof(pms7003::kCmdWakeUp));
  _stream->flush();
}

// Active mode. Default mode after power up. In this mode sensor would send serial data to the host automatically.
void PMS::activeMode()
{
  _stream->write(pms7003::kCmdActive, sizeof(pms7003::kCmdActive));
  _stream->flush();
  _mode = MODE_ACTIVE;
}

// Passive mode. In this mode sensor would send serial data to the host only for request.
void PMS::passiveMode()
{
  uint8_t command[] = { 0x42, 0x4D, 0xE1, 0x00, 0x00, 0x01, 0x70 };
  _stream->write(command, sizeof(command));
  _stream->flush();
  _mode = MODE_PASSIVE;
}

// Request read in Passive Mode.
void PMS::requestRead()
{
  if (_mode == MODE_PASSIVE)
  {
    uint8_t command[] = { 0x42, 0x4D, 0xE2, 0x00, 0x00, 0x01, 0x71 };
    _stream->write(command, sizeof(command));
    _stream->flush();
  }
}

void PMS::reset()
{
  _parser.reset();
}

// Non-blocking function for parse response.
bool PMS::read(DATA& data)
{
  pms7003::PmsFrame frame{};
  bool gotFrame = false;
  while (_stream->available() > 0)
  {
    const uint8_t ch = static_cast<uint8_t>(_stream->read());
    if (_parser.feed(ch, frame))
    {
      data.PM_SP_UG_1_0 = frame.pm1_0_sp;
      data.PM_SP_UG_2_5 = frame.pm2_5_sp;
      data.PM_SP_UG_10_0 = frame.pm10_0_sp;
      data.PM_AE_UG_1_0 = frame.pm1_0_ae;
      data.PM_AE_UG_2_5 = frame.pm2_5_ae;
      data.PM_AE_UG_10_0 = frame.pm10_0_ae;
      gotFrame = true;
    }
  }
  return gotFrame;
}

// Blocking function for parse response. Default timeout is 1s.
bool PMS::readUntil(DATA& data, uint16_t timeout)
{
  uint32_t start = millis();
  do
  {
    if (read(data))
    {
      return true;
    }
  } while (millis() - start < timeout);

  return false;
}

uint8_t PMS::lastErrorCode() const { return _parser.lastErrorCode(); }
uint32_t PMS::sensorErrorCount() const { return _parser.sensorErrorCount(); }
uint32_t PMS::checksumErrorCount() const { return _parser.checksumErrorCount(); }
uint32_t PMS::ackCount() const { return _parser.ackCount(); }

Pms7003Driver::Pms7003Driver(Stream& stream, const pms7003::PmsConfig& config)
  : _stream(stream), _dutyCycle(config) {}

void Pms7003Driver::begin(uint32_t nowMs)
{
  _dutyCycle.begin(nowMs);
}

void Pms7003Driver::sendCommand(const uint8_t cmd[7])
{
  _stream.write(cmd, 7);
  _stream.flush();
}

void Pms7003Driver::flushRx()
{
  while (_stream.available() > 0)
  {
    (void)_stream.read();
  }
}

pms7003::PmsActions Pms7003Driver::poll(uint32_t nowMs)
{
  uint8_t buf[64];
  size_t len = 0;
  while (_stream.available() > 0 && len < sizeof(buf))
  {
    buf[len++] = static_cast<uint8_t>(_stream.read());
  }
  pms7003::PmsActions actions = _dutyCycle.poll(nowMs, buf, len);
  while (_stream.available() > 0 && !actions.flushRx && !actions.sendSleep && !actions.sendWakeUp)
  {
    len = 0;
    while (_stream.available() > 0 && len < sizeof(buf))
    {
      buf[len++] = static_cast<uint8_t>(_stream.read());
    }
    actions = _dutyCycle.poll(nowMs, buf, len);
  }
  if (actions.flushRx)
  {
    flushRx();
  }
  if (actions.sendWakeUp)
  {
    sendCommand(pms7003::kCmdWakeUp);
  }
  if (actions.sendSleep)
  {
    sendCommand(pms7003::kCmdSleep);
  }
  return actions;
}

bool Pms7003Driver::hasPendingReport(uint32_t nowMs)
{
  return _dutyCycle.hasPendingReport(nowMs);
}

const pms7003::PmsFrame& Pms7003Driver::latestFrame() const
{
  return _dutyCycle.latestFrame();
}

void Pms7003Driver::markReportDelivered()
{
  _dutyCycle.markReportDelivered();
}

pms7003::PmsState Pms7003Driver::state() const
{
  return _dutyCycle.state();
}

bool Pms7003Driver::timedOut() const
{
  return _dutyCycle.timedOut();
}

bool Pms7003Driver::hadError() const
{
  return _dutyCycle.hadError();
}

#endif // ARDUINO
