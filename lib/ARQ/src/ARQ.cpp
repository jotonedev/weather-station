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

    status = radio.setRxBandwidth(58);
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

    radio.setTransmitAction(isrHandler, CC1101::GDO0);
    radio.setReceiveAction(isrHandler, CC1101::GDO0);

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

    auto status = radio.startReceive(this->localAddress);
    if (status != CC1101::STATUS_OK) {
        return STATUS_ERROR;
    }

    currentState = RadioState::RECEIVING;
    return STATUS_OK;
}

Status ARQ::poll(uint32_t ackTimeoutMs) {
    switch (currentState) {
        case RadioState::IDLE:
            return STATUS_OK;
        case RadioState::TRANSMITTING:
            return handleStateTransmitting();
        case RadioState::WAITING_ACK:
            return handleStateWaitingAck(ackTimeoutMs);
        case RadioState::SENDING_ACK:
            return handleStateSendingAck();
        case RadioState::RECEIVING:
            return handleStateReceiving();
        default:
            return STATUS_ERROR;
    }
}

Status ARQ::handleStateTransmitting() {
    if (packetDone) {
        radio.finishTransmit();
        packetDone = false;

        // An ACK or NACK doesn't expect an ACK back
        if (currentTxType != DATA_FRAME) {
            radio.idle();
            currentState = RadioState::IDLE;
            return STATUS_OK;
        }

        // Switch to RX mode to wait for ACK
        auto rxStatus = radio.startReceive(this->localAddress);
        if (rxStatus != CC1101::STATUS_OK) {
            radio.idle();
            currentState = RadioState::IDLE;
            return STATUS_ERROR;
        }

        currentState = RadioState::WAITING_ACK;
        opStartTime = millis();
        return STATUS_BUSY;
    }

    // Hardware watchdog guard (e.g. 500 ms at 2 kBaud)
    if (millis() - opStartTime > 500) {
        radio.finishTransmit();
        radio.idle();
        currentState = RadioState::IDLE;
        return STATUS_TIMEOUT;
    }

    return STATUS_BUSY;
}

Status ARQ::handleStateWaitingAck(uint32_t ackTimeoutMs) {
    if (packetDone) {
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
                radio.idle();
                currentState = RadioState::IDLE;
                return STATUS_OK;
            }
        }

        // Not our ACK or packet corrupt; keep listening in RX mode
        radio.startReceive(this->localAddress);
    }

    // Check if timeout for this attempt expired
    if (millis() - opStartTime >= ackTimeoutMs) {
        retryCount++;
        if (retryCount < MAX_RETRIES) {
            // Retransmit saved txBuffer
            packetDone = false;
            opStartTime = millis();
            auto status = radio.startTransmit(txBuffer, this->packetLength, currentTarget);
            if (status != CC1101::STATUS_OK) {
                radio.idle();
                currentState = RadioState::IDLE;
                return STATUS_ERROR;
            }
            currentState = RadioState::TRANSMITTING;
            return STATUS_BUSY;
        } else {
            // All 3 attempts exhausted
            radio.idle();
            currentState = RadioState::IDLE;
            return STATUS_TIMEOUT;
        }
    }

    return STATUS_BUSY;
}

Status ARQ::handleStateSendingAck() {
    if (packetDone) {
        radio.finishTransmit();
        packetDone = false;

        // Return back to listening
        radio.startReceive(this->localAddress);
        currentState = RadioState::RECEIVING;
        return STATUS_OK;
    }

    // Watchdog for ACK transmission
    if (millis() - opStartTime > 500) {
        radio.finishTransmit();
        radio.startReceive(this->localAddress);
        currentState = RadioState::RECEIVING;
        return STATUS_ERROR;
    }

    return STATUS_BUSY;
}

Status ARQ::handleStateReceiving() {
    if (!packetDone) {
        return STATUS_BUSY;
    }

    packetDone = false;
    uint8_t buffer[64];
    size_t bytesRead = 0;

    auto status = radio.readData(buffer, this->packetLength, &bytesRead);
    if (status != CC1101::STATUS_OK || bytesRead < HEADER_SIZE) {
        radio.startReceive(this->localAddress);
        return STATUS_ERROR;
    }

    version_t ver = buffer[0];
    address_t src = buffer[1];
    MessageType type = static_cast<MessageType>(buffer[2]);
    seq_t seq = static_cast<seq_t>(buffer[3]) | (static_cast<seq_t>(buffer[4]) << 8);
    length_t len = static_cast<length_t>(buffer[5]) | (static_cast<length_t>(buffer[6]) << 8);

    if (ver != PROTOCOL_VERSION) {
        radio.startReceive(this->localAddress);
        return STATUS_ERROR;
    }

    if (type == DATA_FRAME) {
        if (len > (this->packetLength - HEADER_SIZE)) {
            radio.startReceive(this->localAddress);
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
    radio.startReceive(this->localAddress);
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

    auto status = radio.startTransmit(txBuffer, this->packetLength, target);
    if (status != CC1101::STATUS_OK) {
        return STATUS_ERROR;
    }

    if (message.type == ACK || message.type == NACK) {
        currentState = RadioState::SENDING_ACK;
    } else {
        currentState = RadioState::TRANSMITTING;
    }

    return STATUS_OK;
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