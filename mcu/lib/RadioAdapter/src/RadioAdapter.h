#ifndef RADIO_ADAPTER_H
#define RADIO_ADAPTER_H

#include <cstddef>
#include <cstdint>

#include <cc1101.h>

#include "RadioInterface.h"

// CC1101RadioAdapter bridges the ARQ RadioInterface to a concrete CC1101::Radio.
// It is the only file that touches the vendored driver and its Arduino types.
class CC1101RadioAdapter : public ARQ::RadioInterface {
  public:
    explicit CC1101RadioAdapter(CC1101::Radio& radio) : radio_(radio) {}

    ARQ::RadioStatus begin() override;
    void setModulation(ARQ::Modulation mod) override;
    ARQ::RadioStatus setFrequency(double megahertz) override;
    ARQ::RadioStatus setFrequencyDeviation(double kilohertz) override;
    ARQ::RadioStatus setRxBandwidth(double kilohertz) override;
    ARQ::RadioStatus setDataRate(double kilobaud) override;
    void setOutputPower(int8_t decibelMilliwatt) override;
    void setPacketLengthMode(ARQ::PacketLengthMode mode, uint8_t length) override;
    void setPacketFormat(ARQ::PacketFormat format) override;
    void setAddressFilteringMode(ARQ::AddressFilteringMode mode) override;
    ARQ::RadioStatus setPreambleLength(uint8_t bytes) override;
    void setSyncWord(uint16_t word) override;
    void setSyncMode(ARQ::SyncMode mode) override;
    void setCrc(bool enabled) override;
    void setDataWhitening(bool enabled) override;
    ARQ::RadioStatus setManchester(bool enabled) override;
    ARQ::RadioStatus setFec(bool enabled) override;
    void setCcaMode(ARQ::CcaMode mode) override;
    void setReceiveAction(void (*handler)(void), ARQ::GdoPin pin) override;
    void writeReg(uint8_t address, uint8_t data) override;
    uint8_t readRegField(uint8_t address, uint8_t highBit, uint8_t lowBit) override;
    ARQ::RadioStatus readData(uint8_t* data, size_t length, size_t* read) override;
    ARQ::RadioStatus startReceive(uint8_t address) override;
    ARQ::RadioStatus transmit(uint8_t* data, size_t length, uint8_t address) override;
    ARQ::RadioStatus transmitWithCca(uint8_t* data, size_t length, uint8_t address,
                                     uint8_t maxAttempts) override;
    void idle() override;
    int8_t getRssi() override;
    uint8_t getLqi() override;
    bool isChannelClear() override;
    bool isCarrierDetected() override;

  private:
    static ARQ::RadioStatus fromRadioStatus(CC1101::Status status);

    CC1101::Radio& radio_;
};

#endif  // RADIO_ADAPTER_H
