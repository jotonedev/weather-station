#include "RadioAdapter.h"

ARQ::RadioStatus CC1101RadioAdapter::fromRadioStatus(CC1101::Status status) {
    switch (status) {
        case CC1101::STATUS_OK:
            return ARQ::RadioStatus::Ok;
        case CC1101::STATUS_INVALID_PARAM:
            return ARQ::RadioStatus::InvalidParam;
        case CC1101::STATUS_CHANNEL_BUSY:
            return ARQ::RadioStatus::ChannelBusy;
        default:
            return ARQ::RadioStatus::Error;
    }
}

ARQ::RadioStatus CC1101RadioAdapter::begin() {
    return fromRadioStatus(radio_.begin());
}

void CC1101RadioAdapter::setModulation(ARQ::Modulation mod) {
    switch (mod) {
        case ARQ::Modulation::Gfsk:
            radio_.setModulation(CC1101::MOD_GFSK);
            break;
    }
}

ARQ::RadioStatus CC1101RadioAdapter::setFrequency(double megahertz) {
    return fromRadioStatus(radio_.setFrequency(megahertz));
}

ARQ::RadioStatus CC1101RadioAdapter::setFrequencyDeviation(double kilohertz) {
    return fromRadioStatus(radio_.setFrequencyDeviation(kilohertz));
}

ARQ::RadioStatus CC1101RadioAdapter::setRxBandwidth(double kilohertz) {
    return fromRadioStatus(radio_.setRxBandwidth(kilohertz));
}

ARQ::RadioStatus CC1101RadioAdapter::setDataRate(double kilobaud) {
    return fromRadioStatus(radio_.setDataRate(kilobaud));
}

void CC1101RadioAdapter::setOutputPower(int8_t decibelMilliwatt) {
    radio_.setOutputPower(decibelMilliwatt);
}

void CC1101RadioAdapter::setPacketLengthMode(ARQ::PacketLengthMode mode, uint8_t length) {
    (void)mode;
    radio_.setPacketLengthMode(CC1101::PKT_LEN_MODE_FIXED, length);
}

void CC1101RadioAdapter::setPacketFormat(ARQ::PacketFormat format) {
    (void)format;
    radio_.setPacketFormat(CC1101::PKT_FORMAT_NORMAL);
}

void CC1101RadioAdapter::setAddressFilteringMode(ARQ::AddressFilteringMode mode) {
    (void)mode;
    radio_.setAddressFilteringMode(CC1101::ADDR_FILTER_MODE_CHECK_BC_0_255);
}

ARQ::RadioStatus CC1101RadioAdapter::setPreambleLength(uint8_t bytes) {
    return fromRadioStatus(radio_.setPreambleLength(bytes));
}

void CC1101RadioAdapter::setSyncWord(uint16_t word) {
    radio_.setSyncWord(word);
}

void CC1101RadioAdapter::setSyncMode(ARQ::SyncMode mode) {
    (void)mode;
    radio_.setSyncMode(CC1101::SYNC_MODE_15_16);
}

void CC1101RadioAdapter::setCrc(bool enabled) {
    radio_.setCrc(enabled);
}

void CC1101RadioAdapter::setDataWhitening(bool enabled) {
    radio_.setDataWhitening(enabled);
}

ARQ::RadioStatus CC1101RadioAdapter::setManchester(bool enabled) {
    return fromRadioStatus(radio_.setManchester(enabled));
}

ARQ::RadioStatus CC1101RadioAdapter::setFec(bool enabled) {
    return fromRadioStatus(radio_.setFEC(enabled));
}

void CC1101RadioAdapter::setCcaMode(ARQ::CcaMode mode) {
    (void)mode;
    radio_.setCcaMode(CC1101::CCA_RSSI_AND_NOT_RX);
}

void CC1101RadioAdapter::setReceiveAction(void (*handler)(void), ARQ::GdoPin pin) {
    (void)pin;
    radio_.setReceiveAction(handler, CC1101::GDO0);
}

void CC1101RadioAdapter::writeReg(uint8_t address, uint8_t data) {
    radio_.writeReg(address, data);
}

uint8_t CC1101RadioAdapter::readRegField(uint8_t address, uint8_t highBit, uint8_t lowBit) {
    return radio_.readRegField(address, highBit, lowBit);
}

ARQ::RadioStatus CC1101RadioAdapter::readData(uint8_t* data, size_t length, size_t* read) {
    return fromRadioStatus(radio_.readData(data, length, read));
}

ARQ::RadioStatus CC1101RadioAdapter::startReceive(uint8_t address) {
    return fromRadioStatus(radio_.startReceive(address));
}

ARQ::RadioStatus CC1101RadioAdapter::transmit(uint8_t* data, size_t length, uint8_t address) {
    return fromRadioStatus(radio_.transmit(data, length, address));
}

ARQ::RadioStatus CC1101RadioAdapter::transmitWithCca(uint8_t* data, size_t length,
                                                     uint8_t address, uint8_t maxAttempts) {
    return fromRadioStatus(radio_.transmitWithCca(data, length, address, maxAttempts));
}

void CC1101RadioAdapter::idle() {
    radio_.idle();
}

int8_t CC1101RadioAdapter::getRssi() {
    return radio_.getRSSI();
}

uint8_t CC1101RadioAdapter::getLqi() {
    return radio_.getLQI();
}

bool CC1101RadioAdapter::isChannelClear() {
    return radio_.isChannelClear();
}

bool CC1101RadioAdapter::isCarrierDetected() {
    return radio_.isCarrierDetected();
}
