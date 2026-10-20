#ifndef ARQ_RADIO_INTERFACE_H
#define ARQ_RADIO_INTERFACE_H

#include <cstddef>
#include <cstdint>

// RadioInterface abstracts the radio hardware behind the ARQ link.
//
// The interface stays free of Arduino and CC1101 dependencies so the ARQ
// state machine compiles and runs on the host for unit tests. The concrete
// driver is wrapped by CC1101RadioAdapter, and tests substitute a mock.
namespace ARQ {

// Radio status values, decoupled from the concrete driver.
enum class RadioStatus : uint8_t { Ok, InvalidParam, ChannelBusy, Error };

// Configuration choices ARQ uses. Each value maps to one CC1101 option.
enum class Modulation { Gfsk };
enum class PacketLengthMode { Fixed };
enum class PacketFormat { Normal };
enum class AddressFilteringMode { CheckBroadcast0255 };
enum class SyncMode { Bits1516 };
enum class CcaMode { RssiAndNotRx };
enum class GdoPin { Gdo0 };

// Register addresses ARQ reads and writes directly.
constexpr uint8_t kRegIocfg0 = 0x02;
constexpr uint8_t kRegRxBytes = 0x3B;

class RadioInterface {
  public:
    virtual ~RadioInterface() = default;

    virtual RadioStatus begin() = 0;
    virtual void setModulation(Modulation mod) = 0;
    virtual RadioStatus setFrequency(double megahertz) = 0;
    virtual RadioStatus setFrequencyDeviation(double kilohertz) = 0;
    virtual RadioStatus setRxBandwidth(double kilohertz) = 0;
    virtual RadioStatus setDataRate(double kilobaud) = 0;
    virtual void setOutputPower(int8_t decibelMilliwatt) = 0;
    virtual void setPacketLengthMode(PacketLengthMode mode, uint8_t length) = 0;
    virtual void setPacketFormat(PacketFormat format) = 0;
    virtual void setAddressFilteringMode(AddressFilteringMode mode) = 0;
    virtual RadioStatus setPreambleLength(uint8_t bytes) = 0;
    virtual void setSyncWord(uint16_t word) = 0;
    virtual void setSyncMode(SyncMode mode) = 0;
    virtual void setCrc(bool enabled) = 0;
    virtual void setDataWhitening(bool enabled) = 0;
    virtual RadioStatus setManchester(bool enabled) = 0;
    virtual RadioStatus setFec(bool enabled) = 0;
    virtual void setCcaMode(CcaMode mode) = 0;
    virtual void setReceiveAction(void (*handler)(void), GdoPin pin) = 0;
    virtual void writeReg(uint8_t address, uint8_t data) = 0;
    virtual uint8_t readRegField(uint8_t address, uint8_t highBit, uint8_t lowBit) = 0;
    virtual RadioStatus readData(uint8_t* data, size_t length, size_t* read) = 0;
    virtual RadioStatus startReceive(uint8_t address) = 0;
    virtual RadioStatus transmit(uint8_t* data, size_t length, uint8_t address) = 0;
    virtual RadioStatus transmitWithCca(uint8_t* data, size_t length, uint8_t address,
                                        uint8_t maxAttempts) = 0;
    virtual void idle() = 0;
    virtual int8_t getRssi() = 0;
    virtual uint8_t getLqi() = 0;
    virtual bool isChannelClear() = 0;
    virtual bool isCarrierDetected() = 0;
};

} // namespace ARQ

#endif // ARQ_RADIO_INTERFACE_H
