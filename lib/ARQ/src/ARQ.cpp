#include <Arduino.h>
#include <cc1101.h>

#include "ARQ.h"

namespace ARQ {

ARQ* ARQ::instance = nullptr;

ARQ::~ARQ() {
    delete[] txBuffer;
    delete[] rxPayloadBuffer;
    txBuffer = nullptr;
    rxPayloadBuffer = nullptr;
}

Status ARQ::initializeCC1101(uint8_t packetLength) {
    instance = this;
    this->packetLength = packetLength - 1; // Subtract 1 for the address byte in the packet
    this->hasNewMessage = false;
    this->hasLastReceived = false;
    this->currentTxSeq = 0;

    delete[] txBuffer;
    delete[] rxPayloadBuffer;
    txBuffer = new uint8_t[this->packetLength];
    rxPayloadBuffer = new uint8_t[this->packetLength];
    memset(txBuffer, 0, this->packetLength);
    memset(rxPayloadBuffer, 0, this->packetLength);

    auto status = radio.begin();
    if (status != CC1101::STATUS_OK) {
        return STATUS_CHIP_NOT_FOUND;
    }

    radio.setModulation(CC1101::MOD_GFSK);
    status = radio.setFrequency(433.5);
    if (status != CC1101::STATUS_OK) {
        return STATUS_INVALID_PARAM;
    }
    status = radio.setFrequencyDeviation(5.0);
    if (status != CC1101::STATUS_OK) return STATUS_INVALID_PARAM;

    status = radio.setRxBandwidth(100);
    if (status != CC1101::STATUS_OK) return STATUS_INVALID_PARAM;
    status = radio.setDataRate(2.0);  // 2 kBaud
    if (status != CC1101::STATUS_OK) return STATUS_INVALID_PARAM;
    radio.setOutputPower(10);  // 10 dBm

    radio.setPacketLengthMode(CC1101::PKT_LEN_MODE_FIXED, packetLength);
    radio.setPacketFormat(CC1101::PKT_FORMAT_NORMAL);
    radio.setAddressFilteringMode(CC1101::ADDR_FILTER_MODE_CHECK_BC_0_255);

    status = radio.setPreambleLength(64);
    if (status != CC1101::STATUS_OK) return STATUS_INVALID_PARAM;
    radio.setSyncWord(0xD391);
    radio.setSyncMode(CC1101::SYNC_MODE_15_16);

    radio.setCrc(true);
    radio.setDataWhitening(true);
    status = radio.setManchester(false);
    if (status != CC1101::STATUS_OK) return STATUS_INVALID_PARAM;
    status = radio.setFEC(true);
    if (status != CC1101::STATUS_OK) return STATUS_INVALID_PARAM;

    radio.setCcaMode(CC1101::CCA_RSSI_AND_NOT_RX);

    radio.setReceiveAction(isrHandler, CC1101::GDO0);
    radio.writeReg(CC1101_REG_IOCFG0, 0x07);

    currentState = RadioState::IDLE;

    return STATUS_OK;
}

Status ARQ::startSendData(const byte* data, length_t length, address_t target) {
    if (HEADER_SIZE + length > this->packetLength) {
        return STATUS_INVALID_SIZE;
    }

    // If currently listening, gracefully put radio to idle first
    if (currentState == RadioState::RECEIVING) {
        radio.idle();
        currentState = RadioState::IDLE;
    }

    if (currentState != RadioState::IDLE) {
        return STATUS_BUSY;
    }

    currentTxSeq++;
    txStartTime = millis();
    Message message {
        .version = PROTOCOL_VERSION,
        .source = this->localAddress,
        .type = DATA_FRAME,
        .sequence = currentTxSeq,
        .length = length,
        .payload = data
    };
    return this->sendMessage(message, target);
}

Status ARQ::enterRxMode() {
    packetDone = false;
    auto status = radio.startReceive(this->localAddress);
    return (status == CC1101::STATUS_OK) ? STATUS_OK : STATUS_ERROR;
}

Status ARQ::startReceiveData(address_t localAddress) {
    this->localAddress = localAddress;
    return startReceiveData();
}

Status ARQ::startReceiveData() {
    if (currentState != RadioState::IDLE) {
        return STATUS_BUSY;
    }

    packetDone = false;
    opStartTime = millis();

    radio.setReceiveAction(isrHandler, CC1101::GDO0);
    auto status = enterRxMode();
    if (status != STATUS_OK) {
        return STATUS_ERROR;
    }

    currentState = RadioState::RECEIVING;
    return STATUS_OK;
}

Status ARQ::poll(uint32_t ackTimeoutMs) {
    switch (currentState) {
        case RadioState::IDLE:
            return STATUS_OK;
        case RadioState::WAITING_ACK:
            return handleStateWaitingAck(ackTimeoutMs);
        case RadioState::RECEIVING:
            return handleStateReceiving();
        default:
            return STATUS_OK;
    }
}

Status ARQ::handleStateTransmitting() {
    return STATUS_OK;
}

Status ARQ::handleStateWaitingAck(uint32_t ackTimeoutMs) {
    uint8_t rxBytes = radio.readRegField(CC1101_REG_RXBYTES, 6, 0);
    if (packetDone || rxBytes >= (this->packetLength + 1)) {
        packetDone = false;
        uint8_t buffer[64];
        size_t bytesRead = 0;

        auto status = radio.readData(buffer, this->packetLength, &bytesRead);
        if (status == CC1101::STATUS_OK && bytesRead >= HEADER_SIZE) {
            version_t ver = buffer[0];
            address_t src = buffer[1];
            MessageType type = static_cast<MessageType>(buffer[2]);
            seq_t seq = static_cast<seq_t>(buffer[3]) | (static_cast<seq_t>(buffer[4]) << 8);

            if (ver == PROTOCOL_VERSION && type == ACK && src == currentTarget && seq == currentTxSeq) {
                lastRTT = millis() - txStartTime;
                txSuccessCount++;
                radio.idle();
                currentState = RadioState::IDLE;
                return STATUS_OK;
            }
        }

        // Not our ACK or packet corrupt; keep listening in RX mode
        enterRxMode();
    }

    // Check if timeout for this attempt expired
    if (millis() - opStartTime >= ackTimeoutMs) {
        retryCount++;
        totalRetries++;
        if (retryCount < MAX_RETRIES) {
            // Retransmit saved txBuffer synchronously (with CCA if enabled)
            CC1101::Status status;
            if (ccaEnabled) {
                status = radio.transmitWithCca(txBuffer, this->packetLength, currentTarget, 3);
            } else {
                status = radio.transmit(txBuffer, this->packetLength, currentTarget);
            }
            if (status != CC1101::STATUS_OK) {
                if (status == CC1101::STATUS_CHANNEL_BUSY) {
                    enterRxMode();
                    currentState = RadioState::WAITING_ACK;
                    return STATUS_BUSY;
                }
                radio.idle();
                currentState = RadioState::IDLE;
                return STATUS_ERROR;
            }
            packetDone = false;
            opStartTime = millis();
            enterRxMode();
            currentState = RadioState::WAITING_ACK;
            return STATUS_BUSY;
        } else {
            // All 3 attempts exhausted
            txTimeoutCount++;
            radio.idle();
            currentState = RadioState::IDLE;
            return STATUS_TIMEOUT;
        }
    }

    return STATUS_BUSY;
}

Status ARQ::handleStateSendingAck() {
    return STATUS_OK;
}

Status ARQ::handleStateReceiving() {
    uint8_t rxBytes = radio.readRegField(CC1101_REG_RXBYTES, 6, 0);
    if (!packetDone && rxBytes < (this->packetLength + 1)) {
        return STATUS_BUSY;
    }

    packetDone = false;
    uint8_t buffer[64];
    size_t bytesRead = 0;

    auto status = radio.readData(buffer, this->packetLength, &bytesRead);
    if (status != CC1101::STATUS_OK || bytesRead < HEADER_SIZE) {
        enterRxMode();
        return STATUS_ERROR;
    }

    lastRssi = radio.getRSSI();
    lastLqi = radio.getLQI();

    version_t ver = buffer[0];
    address_t src = buffer[1];
    MessageType type = static_cast<MessageType>(buffer[2]);
    seq_t seq = static_cast<seq_t>(buffer[3]) | (static_cast<seq_t>(buffer[4]) << 8);
    length_t len = static_cast<length_t>(buffer[5]) | (static_cast<length_t>(buffer[6]) << 8);

    if (ver != PROTOCOL_VERSION) {
        enterRxMode();
        return STATUS_ERROR;
    }

    if (type == DATA_FRAME) {
        if (len > (this->packetLength - HEADER_SIZE)) {
            enterRxMode();
            return STATUS_INVALID_SIZE;
        }

        // Check for duplicate packet
        bool isDuplicate = (hasLastReceived && lastReceivedSource == src && lastReceivedSeq == seq);
        if (!isDuplicate) {
            rxMessage.version = ver;
            rxMessage.source = src;
            rxMessage.type = type;
            rxMessage.sequence = seq;
            rxMessage.length = len;
            if (len > 0) {
                memcpy(rxPayloadBuffer, buffer + HEADER_SIZE, len);
            }
            rxMessage.payload = rxPayloadBuffer;
            hasNewMessage = true;

            lastReceivedSource = src;
            lastReceivedSeq = seq;
            hasLastReceived = true;
        }

        // Reply with ACK to the sender
        sendACK(src, seq);
        return STATUS_OK;
    }

    // Other packet types: ignore and keep receiving
    enterRxMode();
    return STATUS_BUSY;
}

Status ARQ::readMessage(Message& outMessage, byte* outPayloadBuffer, length_t maxLen) {
    if (!hasNewMessage) {
        return STATUS_BUSY;
    }

    outMessage = rxMessage;
    if (outPayloadBuffer != nullptr && maxLen >= rxMessage.length) {
        if (rxMessage.length > 0) {
            memcpy(outPayloadBuffer, rxPayloadBuffer, rxMessage.length);
        }
        outMessage.payload = outPayloadBuffer;
    } else {
        outMessage.payload = rxPayloadBuffer;
    }

    hasNewMessage = false;
    return STATUS_OK;
}

Status ARQ::sendACK(address_t target, seq_t sequence) {
    Message ackMessage {
        .version = PROTOCOL_VERSION,
        .source = this->localAddress,
        .type = ACK,
        .sequence = sequence,
        .length = 0,
        .payload = nullptr
    };
    return this->sendMessage(ackMessage, target);
}

Status ARQ::sendNACK(address_t target, seq_t sequence) {
    Message nackMessage {
        .version = PROTOCOL_VERSION,
        .source = this->localAddress,
        .type = NACK,
        .sequence = sequence,
        .length = 0,
        .payload = nullptr
    };
    return this->sendMessage(nackMessage, target);
}

Status ARQ::sendMessage(Message& message, address_t target) {
    if (HEADER_SIZE + message.length > this->packetLength) {
        return STATUS_INVALID_SIZE;
    }

    if (currentState != RadioState::IDLE && currentState != RadioState::RECEIVING) {
        return STATUS_BUSY;
    }

    packetDone = false;
    opStartTime = millis();
    currentTxType = message.type;
    currentTarget = target;

    txBuffer[0] = message.version;
    txBuffer[1] = message.source;
    txBuffer[2] = static_cast<uint8_t>(message.type);
    txBuffer[3] = static_cast<uint8_t>(message.sequence & 0xFF);
    txBuffer[4] = static_cast<uint8_t>((message.sequence >> 8) & 0xFF);
    txBuffer[5] = static_cast<uint8_t>(message.length & 0xFF);
    txBuffer[6] = static_cast<uint8_t>((message.length >> 8) & 0xFF);

    if (message.length > 0 && message.payload != nullptr) {
        memcpy(txBuffer + HEADER_SIZE, message.payload, message.length);
    }
    if (this->packetLength > HEADER_SIZE + message.length) {
        memset(txBuffer + HEADER_SIZE + message.length, 0, this->packetLength - (HEADER_SIZE + message.length));
    }

    if (message.type == DATA_FRAME) {
        retryCount = 0;
    }

    CC1101::Status status;
    if (ccaEnabled && message.type == DATA_FRAME) {
        status = radio.transmitWithCca(txBuffer, this->packetLength, target, 5);
    } else {
        status = radio.transmit(txBuffer, this->packetLength, target);
    }

    if (status != CC1101::STATUS_OK) {
        if (status == CC1101::STATUS_CHANNEL_BUSY) {
            enterRxMode();
            return STATUS_CHANNEL_BUSY;
        }
        radio.idle();
        currentState = RadioState::IDLE;
        return STATUS_ERROR;
    }

    if (message.type == ACK || message.type == NACK) {
        packetDone = false;
        enterRxMode();
        currentState = RadioState::RECEIVING;
        return STATUS_OK;
    } else {
        packetDone = false;
        opStartTime = millis();
        auto rxStatus = enterRxMode();
        if (rxStatus != STATUS_OK) {
            radio.idle();
            currentState = RadioState::IDLE;
            return STATUS_ERROR;
        }
        currentState = RadioState::WAITING_ACK;
        return STATUS_OK;
    }
}

void ARQ::isrHandler() {
    if (instance) {
        instance->handleInterrupt();
    }
}

void ARQ::handleInterrupt() {
    packetDone = true;
}

}  // namespace ARQ