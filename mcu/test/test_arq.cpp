#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include <chrono>
#include <cstdint>
#include <cstring>
#include <thread>

#include "ARQ.h"

using ::testing::_; // NOLINT(bugprone-reserved-identifier)/gtest matcher convention
using ::testing::Invoke;
using ::testing::NiceMock;
using ::testing::Return;

namespace {

class MockRadio : public ARQ::RadioInterface {
  public:
    MOCK_METHOD(ARQ::RadioStatus, begin, (), (override));
    MOCK_METHOD(void, setModulation, (ARQ::Modulation), (override));
    MOCK_METHOD(ARQ::RadioStatus, setFrequency, (double), (override));
    MOCK_METHOD(ARQ::RadioStatus, setFrequencyDeviation, (double), (override));
    MOCK_METHOD(ARQ::RadioStatus, setRxBandwidth, (double), (override));
    MOCK_METHOD(ARQ::RadioStatus, setDataRate, (double), (override));
    MOCK_METHOD(void, setOutputPower, (int8_t), (override));
    MOCK_METHOD(void, setPacketLengthMode, (ARQ::PacketLengthMode, uint8_t), (override));
    MOCK_METHOD(void, setPacketFormat, (ARQ::PacketFormat), (override));
    MOCK_METHOD(void, setAddressFilteringMode, (ARQ::AddressFilteringMode), (override));
    MOCK_METHOD(ARQ::RadioStatus, setPreambleLength, (uint8_t), (override));
    MOCK_METHOD(void, setSyncWord, (uint16_t), (override));
    MOCK_METHOD(void, setSyncMode, (ARQ::SyncMode), (override));
    MOCK_METHOD(void, setCrc, (bool), (override));
    MOCK_METHOD(void, setDataWhitening, (bool), (override));
    MOCK_METHOD(ARQ::RadioStatus, setManchester, (bool), (override));
    MOCK_METHOD(ARQ::RadioStatus, setFec, (bool), (override));
    MOCK_METHOD(void, setCcaMode, (ARQ::CcaMode), (override));
    MOCK_METHOD(void, setReceiveAction, (void (*)(), ARQ::GdoPin), (override));
    MOCK_METHOD(void, writeReg, (uint8_t, uint8_t), (override));
    MOCK_METHOD(uint8_t, readRegField, (uint8_t, uint8_t, uint8_t), (override));
    MOCK_METHOD(ARQ::RadioStatus, readData, (uint8_t*, size_t, size_t*), (override));
    MOCK_METHOD(ARQ::RadioStatus, startReceive, (uint8_t), (override));
    MOCK_METHOD(ARQ::RadioStatus, transmit, (uint8_t*, size_t, uint8_t), (override));
    MOCK_METHOD(ARQ::RadioStatus, transmitWithCca, (uint8_t*, size_t, uint8_t, uint8_t),
                (override));
    MOCK_METHOD(void, idle, (), (override));
    MOCK_METHOD(int8_t, getRssi, (), (override));
    MOCK_METHOD(uint8_t, getLqi, (), (override));
    MOCK_METHOD(bool, isChannelClear, (), (override));
    MOCK_METHOD(bool, isCarrierDetected, (), (override));
};

// Fill a frame header and sequential 1-based payload bytes into the caller buffer.
void fillFrame(uint8_t* data, size_t* read, ARQ::address_t src, ARQ::MessageType type,
               ARQ::seq_t seq, ARQ::length_t payloadLen) {
    data[0] = ARQ::PROTOCOL_VERSION;
    data[1] = src;
    data[2] = static_cast<uint8_t>(type);
    data[3] = static_cast<uint8_t>(seq & 0xFF);
    data[4] = static_cast<uint8_t>((seq >> 8) & 0xFF);
    data[5] = static_cast<uint8_t>(payloadLen & 0xFF);
    data[6] = static_cast<uint8_t>((payloadLen >> 8) & 0xFF);
    for (ARQ::length_t i = 0; i < payloadLen; ++i) {
        data[ARQ::HEADER_SIZE + i] = static_cast<uint8_t>(i + 1);
    }
    *read = static_cast<size_t>(ARQ::HEADER_SIZE) + payloadLen;
}

// Fill a header plus 4-byte payload into the caller buffer, as a received frame.
void fillDataFrame(uint8_t* data, size_t length, size_t* read, ARQ::seq_t seq) {
    (void)length;
    fillFrame(data, read, 1, ARQ::DATA_FRAME, seq, 4);
}

} // namespace

TEST(Arq, InitializesRadioConfiguration) {
    NiceMock<MockRadio> mock;
    EXPECT_CALL(mock, begin()).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, setFrequency(433.5)).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, setPacketLengthMode(ARQ::PacketLengthMode::Fixed, 48));
    EXPECT_CALL(mock, setReceiveAction(_, ARQ::GdoPin::Gdo0));

    ARQ::ARQ arq(mock, 1);
    EXPECT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);
    EXPECT_EQ(arq.getState(), ARQ::RadioState::IDLE);
}

TEST(Arq, InitializeRejectsMissingChip) {
    NiceMock<MockRadio> mock;
    EXPECT_CALL(mock, begin()).WillOnce(Return(ARQ::RadioStatus::Error));

    ARQ::ARQ arq(mock, 1);
    EXPECT_EQ(arq.initializeCC1101(), ARQ::STATUS_CHIP_NOT_FOUND);
}

TEST(Arq, SendHappyPathWithAck) {
    NiceMock<MockRadio> mock;
    ARQ::ARQ arq(mock, 1);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    EXPECT_CALL(mock, transmitWithCca(_, _, 2, 5)).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, startReceive(1)).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, readRegField(ARQ::kRegRxBytes, 6, 0)).WillOnce(Return(48));
    EXPECT_CALL(mock, readData(_, _, _))
        .WillOnce(Invoke([](uint8_t* data, size_t length, size_t* read) {
            (void)length;
            data[0] = ARQ::PROTOCOL_VERSION;
            data[1] = 2; // source is the base station
            data[2] = ARQ::ACK;
            data[3] = 1; // first sequence number
            data[4] = 0;
            data[5] = 0;
            data[6] = 0;
            *read = 7;
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_CALL(mock, idle()).Times(1);

    uint8_t payload[4] = {1, 2, 3, 4};
    EXPECT_EQ(arq.startSendData(payload, 4, 2), ARQ::STATUS_OK);
    EXPECT_EQ(arq.poll(1000), ARQ::STATUS_OK);
    EXPECT_EQ(arq.getState(), ARQ::RadioState::IDLE);
    EXPECT_EQ(arq.getTxSuccessCount(), 1);
}

TEST(Arq, SendRetriesThenTimesOut) {
    NiceMock<MockRadio> mock;
    ARQ::ARQ arq(mock, 1);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    // Initial send plus two retransmissions exhaust the retry budget.
    EXPECT_CALL(mock, transmitWithCca(_, _, 2, _))
        .Times(3)
        .WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, startReceive(1)).Times(3).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, idle()).Times(1);

    uint8_t payload[4] = {1, 2, 3, 4};
    ASSERT_EQ(arq.startSendData(payload, 4, 2), ARQ::STATUS_OK);

    ARQ::Status result = ARQ::STATUS_OK;
    int guard = 0;
    while (arq.getState() != ARQ::RadioState::IDLE && guard < 5000) {
        std::this_thread::sleep_for(std::chrono::milliseconds(2));
        result = arq.poll(5);
        guard++;
    }

    EXPECT_EQ(result, ARQ::STATUS_TIMEOUT);
    EXPECT_EQ(arq.getTxTimeoutCount(), 1);
    EXPECT_EQ(arq.getTxRetryCount(), 3);
    EXPECT_EQ(arq.getState(), ARQ::RadioState::IDLE);
}

TEST(Arq, SendReportsChannelBusy) {
    NiceMock<MockRadio> mock;
    ARQ::ARQ arq(mock, 1);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    EXPECT_CALL(mock, transmitWithCca(_, _, _, _)).WillOnce(Return(ARQ::RadioStatus::ChannelBusy));
    EXPECT_CALL(mock, startReceive(1)).WillOnce(Return(ARQ::RadioStatus::Ok));

    uint8_t payload[4] = {1, 2, 3, 4};
    EXPECT_EQ(arq.startSendData(payload, 4, 2), ARQ::STATUS_CHANNEL_BUSY);
}

TEST(Arq, SendRejectsOversizePayload) {
    NiceMock<MockRadio> mock;
    EXPECT_CALL(mock, transmitWithCca(_, _, _, _)).Times(0);
    ARQ::ARQ arq(mock, 1);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    uint8_t payload[42] = {0};
    EXPECT_EQ(arq.startSendData(payload, 42, 2), ARQ::STATUS_INVALID_SIZE);
}

TEST(Arq, ReceiveDeliversDataAndAcks) {
    NiceMock<MockRadio> mock;
    ARQ::ARQ arq(mock, 2);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    EXPECT_CALL(mock, startReceive(2)).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, readRegField(ARQ::kRegRxBytes, 6, 0)).WillRepeatedly(Return(48));
    EXPECT_CALL(mock, readData(_, _, _))
        .WillRepeatedly(Invoke([](uint8_t* data, size_t length, size_t* read) {
            fillDataFrame(data, length, read, 5);
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_CALL(mock, getRssi()).WillRepeatedly(Return(-60));
    EXPECT_CALL(mock, getLqi()).WillRepeatedly(Return(30));
    EXPECT_CALL(mock, transmit(_, _, _)).WillRepeatedly(Return(ARQ::RadioStatus::Ok));

    EXPECT_EQ(arq.startReceiveData(2), ARQ::STATUS_OK);
    EXPECT_EQ(arq.poll(), ARQ::STATUS_OK);
    ASSERT_TRUE(arq.available());

    ARQ::Message message;
    uint8_t buffer[16];
    EXPECT_EQ(arq.readMessage(message, buffer, sizeof(buffer)), ARQ::STATUS_OK);
    EXPECT_EQ(message.length, 4);
    EXPECT_EQ(message.source, 1);
    EXPECT_EQ(message.sequence, 5);
    EXPECT_EQ(std::memcmp(buffer, "\x01\x02\x03\x04", 4), 0);
    EXPECT_EQ(arq.getLastRSSI(), -60);
    EXPECT_EQ(arq.getLastLQI(), 30);
}

TEST(Arq, ReceiveSuppressesDuplicate) {
    NiceMock<MockRadio> mock;
    ARQ::ARQ arq(mock, 2);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    EXPECT_CALL(mock, startReceive(2)).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, readRegField(ARQ::kRegRxBytes, 6, 0)).WillRepeatedly(Return(48));
    EXPECT_CALL(mock, readData(_, _, _))
        .WillRepeatedly(Invoke([](uint8_t* data, size_t length, size_t* read) {
            fillDataFrame(data, length, read, 5);
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_CALL(mock, transmit(_, _, _)).WillRepeatedly(Return(ARQ::RadioStatus::Ok));

    EXPECT_EQ(arq.startReceiveData(2), ARQ::STATUS_OK);
    EXPECT_EQ(arq.poll(), ARQ::STATUS_OK);
    ASSERT_TRUE(arq.available());
    ARQ::Message message;
    uint8_t buffer[16];
    EXPECT_EQ(arq.readMessage(message, buffer, sizeof(buffer)), ARQ::STATUS_OK);

    // The same frame arrives again and is dropped, no new message surfaces.
    EXPECT_EQ(arq.poll(), ARQ::STATUS_OK);
    EXPECT_FALSE(arq.available());
}

TEST(Arq, ReceiveRejectsWrongVersion) {
    NiceMock<MockRadio> mock;
    ARQ::ARQ arq(mock, 2);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    EXPECT_CALL(mock, startReceive(2)).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, readRegField(ARQ::kRegRxBytes, 6, 0)).WillOnce(Return(48));
    EXPECT_CALL(mock, readData(_, _, _))
        .WillOnce(Invoke([](uint8_t* data, size_t length, size_t* read) {
            (void)length;
            data[0] = 2; // unknown protocol version
            data[1] = 1;
            data[2] = ARQ::DATA_FRAME;
            data[3] = 1;
            data[4] = 0;
            data[5] = 4;
            data[6] = 0;
            *read = 11;
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_CALL(mock, transmit(_, _, _)).Times(0);

    EXPECT_EQ(arq.startReceiveData(2), ARQ::STATUS_OK);
    EXPECT_EQ(arq.poll(), ARQ::STATUS_ERROR);
    EXPECT_FALSE(arq.available());
}

TEST(Arq, InitializeRejectsInvalidPacketLength) {
    NiceMock<MockRadio> invalidMock;
    EXPECT_CALL(invalidMock, begin()).Times(0);

    ARQ::ARQ invalidArq(invalidMock, 1);
    EXPECT_EQ(invalidArq.initializeCC1101(0), ARQ::STATUS_INVALID_PARAM);
    EXPECT_EQ(invalidArq.initializeCC1101(7), ARQ::STATUS_INVALID_PARAM);
    EXPECT_EQ(invalidArq.initializeCC1101(66), ARQ::STATUS_INVALID_PARAM);

    NiceMock<MockRadio> minMock;
    EXPECT_CALL(minMock, begin()).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(minMock, setPacketLengthMode(ARQ::PacketLengthMode::Fixed, 8));
    ARQ::ARQ minArq(minMock, 1);
    EXPECT_EQ(minArq.initializeCC1101(8), ARQ::STATUS_OK);

    NiceMock<MockRadio> maxMock;
    EXPECT_CALL(maxMock, begin()).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(maxMock, setPacketLengthMode(ARQ::PacketLengthMode::Fixed, 65));
    ARQ::ARQ maxArq(maxMock, 1);
    EXPECT_EQ(maxArq.initializeCC1101(65), ARQ::STATUS_OK);
}

TEST(Arq, InitializeReturnsInvalidParamOnRadioConfigFailure) {
    for (int failStep = 0; failStep < 7; ++failStep) {
        NiceMock<MockRadio> mock;
        EXPECT_CALL(mock, begin()).WillOnce(Return(ARQ::RadioStatus::Ok));
        EXPECT_CALL(mock, setReceiveAction(_, _)).Times(0);

        auto statusFor = [failStep](int step) {
            return (step == failStep) ? ARQ::RadioStatus::InvalidParam : ARQ::RadioStatus::Ok;
        };
        ON_CALL(mock, setFrequency(_)).WillByDefault(Return(statusFor(0)));
        ON_CALL(mock, setFrequencyDeviation(_)).WillByDefault(Return(statusFor(1)));
        ON_CALL(mock, setRxBandwidth(_)).WillByDefault(Return(statusFor(2)));
        ON_CALL(mock, setDataRate(_)).WillByDefault(Return(statusFor(3)));
        ON_CALL(mock, setPreambleLength(_)).WillByDefault(Return(statusFor(4)));
        ON_CALL(mock, setManchester(_)).WillByDefault(Return(statusFor(5)));
        ON_CALL(mock, setFec(_)).WillByDefault(Return(statusFor(6)));

        ARQ::ARQ arq(mock, 1);
        EXPECT_EQ(arq.initializeCC1101(), ARQ::STATUS_INVALID_PARAM) << "failStep=" << failStep;
    }
}

TEST(Arq, StartSendDataBoundary40Vs41Payload) {
    NiceMock<MockRadio> okMock;
    ARQ::ARQ okArq(okMock, 1);
    ASSERT_EQ(okArq.initializeCC1101(), ARQ::STATUS_OK);

    uint8_t encodedLenLow = 0;
    uint8_t encodedLenHigh = 0xFF;
    EXPECT_CALL(okMock, transmitWithCca(_, 47, 2, 5))
        .WillOnce(Invoke([&](uint8_t* data, size_t length, uint8_t target, uint8_t maxRetries) {
            (void)length;
            (void)target;
            (void)maxRetries;
            encodedLenLow = data[5];
            encodedLenHigh = data[6];
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_CALL(okMock, startReceive(1)).WillOnce(Return(ARQ::RadioStatus::Ok));

    uint8_t payload[41] = {0};
    EXPECT_EQ(okArq.startSendData(payload, 40, 2), ARQ::STATUS_OK);
    EXPECT_EQ(encodedLenLow, 40);
    EXPECT_EQ(encodedLenHigh, 0);
    EXPECT_EQ(okArq.getState(), ARQ::RadioState::WAITING_ACK);

    NiceMock<MockRadio> errMock;
    EXPECT_CALL(errMock, transmitWithCca(_, _, _, _)).Times(0);
    ARQ::ARQ errArq(errMock, 1);
    ASSERT_EQ(errArq.initializeCC1101(), ARQ::STATUS_OK);

    EXPECT_EQ(errArq.startSendData(payload, 41, 2), ARQ::STATUS_INVALID_SIZE);
    EXPECT_EQ(errArq.getState(), ARQ::RadioState::IDLE);
    EXPECT_EQ(errArq.startSendData(nullptr, 4, 2), ARQ::STATUS_INVALID_PARAM);
    EXPECT_EQ(errArq.getState(), ARQ::RadioState::IDLE);
}

TEST(Arq, StartSendDataBusyInWaitingAckAndPreemptsReceiving) {
    NiceMock<MockRadio> mock;
    ARQ::ARQ arq(mock, 1);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    EXPECT_CALL(mock, transmitWithCca(_, 47, 2, 5)).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, startReceive(1)).WillOnce(Return(ARQ::RadioStatus::Ok));
    uint8_t payload[4] = {1, 2, 3, 4};
    ASSERT_EQ(arq.startSendData(payload, 4, 2), ARQ::STATUS_OK);
    ASSERT_EQ(arq.getState(), ARQ::RadioState::WAITING_ACK);

    EXPECT_CALL(mock, idle()).Times(0);
    EXPECT_CALL(mock, transmitWithCca(_, _, _, _)).Times(0);
    EXPECT_EQ(arq.startSendData(payload, 4, 2), ARQ::STATUS_BUSY);
    EXPECT_EQ(arq.getState(), ARQ::RadioState::WAITING_ACK);

    NiceMock<MockRadio> rxMock;
    ARQ::ARQ rxArq(rxMock, 1);
    ASSERT_EQ(rxArq.initializeCC1101(), ARQ::STATUS_OK);
    EXPECT_CALL(rxMock, startReceive(1)).Times(2).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    ASSERT_EQ(rxArq.startReceiveData(1), ARQ::STATUS_OK);
    ASSERT_EQ(rxArq.getState(), ARQ::RadioState::RECEIVING);

    EXPECT_CALL(rxMock, idle()).Times(1);
    EXPECT_CALL(rxMock, transmitWithCca(_, 47, 2, 5)).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_EQ(rxArq.startSendData(payload, 4, 2), ARQ::STATUS_OK);
    EXPECT_EQ(rxArq.getState(), ARQ::RadioState::WAITING_ACK);

    NiceMock<MockRadio> busyMock;
    ARQ::ARQ busyArq(busyMock, 1);
    ASSERT_EQ(busyArq.initializeCC1101(), ARQ::STATUS_OK);
    EXPECT_CALL(busyMock, startReceive(1)).Times(2).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    ASSERT_EQ(busyArq.startReceiveData(1), ARQ::STATUS_OK);
    EXPECT_CALL(busyMock, idle()).Times(1);
    EXPECT_CALL(busyMock, transmitWithCca(_, 47, 2, 5))
        .WillOnce(Return(ARQ::RadioStatus::ChannelBusy));
    EXPECT_EQ(busyArq.startSendData(payload, 4, 2), ARQ::STATUS_CHANNEL_BUSY);
    EXPECT_EQ(busyArq.getState(), ARQ::RadioState::IDLE);
}

TEST(Arq, StartReceiveDataBusyAndError) {
    NiceMock<MockRadio> mock;
    ARQ::ARQ arq(mock, 1);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    EXPECT_CALL(mock, startReceive(2)).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_EQ(arq.startReceiveData(2), ARQ::STATUS_OK);
    EXPECT_EQ(arq.getState(), ARQ::RadioState::RECEIVING);
    EXPECT_EQ(arq.getLocalAddress(), 2);

    EXPECT_CALL(mock, startReceive(_)).Times(0);
    EXPECT_EQ(arq.startReceiveData(3), ARQ::STATUS_BUSY);
    EXPECT_EQ(arq.getLocalAddress(), 2);

    NiceMock<MockRadio> waitMock;
    ARQ::ARQ waitArq(waitMock, 1);
    ASSERT_EQ(waitArq.initializeCC1101(), ARQ::STATUS_OK);
    EXPECT_CALL(waitMock, transmitWithCca(_, 47, 2, 5)).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(waitMock, startReceive(1)).WillOnce(Return(ARQ::RadioStatus::Ok));
    uint8_t payload[2] = {0xAA, 0xBB};
    ASSERT_EQ(waitArq.startSendData(payload, 2, 2), ARQ::STATUS_OK);
    EXPECT_EQ(waitArq.startReceiveData(), ARQ::STATUS_BUSY);

    NiceMock<MockRadio> errMock;
    ARQ::ARQ errArq(errMock, 1);
    ASSERT_EQ(errArq.initializeCC1101(), ARQ::STATUS_OK);
    EXPECT_CALL(errMock, startReceive(2)).WillOnce(Return(ARQ::RadioStatus::Error));
    EXPECT_EQ(errArq.startReceiveData(2), ARQ::STATUS_ERROR);
    EXPECT_EQ(errArq.getState(), ARQ::RadioState::IDLE);
}

TEST(Arq, WaitingAckIgnoresMismatchedAndNonAckFrames) {
    NiceMock<MockRadio> mock;
    ARQ::ARQ arq(mock, 1);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    EXPECT_CALL(mock, transmitWithCca(_, 47, 2, 5)).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, startReceive(1)).Times(5).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, readRegField(ARQ::kRegRxBytes, 6, 0)).Times(5).WillRepeatedly(Return(48));

    int callIndex = 0;
    EXPECT_CALL(mock, readData(_, _, _))
        .Times(5)
        .WillRepeatedly(Invoke([&](uint8_t* data, size_t length, size_t* read) {
            (void)length;
            switch (callIndex++) {
            case 0:
                fillFrame(data, read, 3, ARQ::ACK, 1, 0); // wrong source
                break;
            case 1:
                fillFrame(data, read, 2, ARQ::ACK, 2, 0); // wrong sequence
                break;
            case 2:
                fillFrame(data, read, 2, ARQ::DATA_FRAME, 1, 0); // wrong type: DATA_FRAME
                break;
            case 3:
                fillFrame(data, read, 2, ARQ::NACK, 1, 0); // wrong type: NACK
                break;
            default:
                fillFrame(data, read, 2, ARQ::ACK, 1, 0); // matching ACK
                break;
            }
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_CALL(mock, idle()).Times(1);

    uint8_t payload[4] = {1, 2, 3, 4};
    ASSERT_EQ(arq.startSendData(payload, 4, 2), ARQ::STATUS_OK);

    for (int i = 0; i < 4; ++i) {
        EXPECT_EQ(arq.poll(1000), ARQ::STATUS_BUSY);
        EXPECT_EQ(arq.getState(), ARQ::RadioState::WAITING_ACK);
        EXPECT_EQ(arq.getTxSuccessCount(), 0u);
    }

    EXPECT_EQ(arq.poll(1000), ARQ::STATUS_OK);
    EXPECT_EQ(arq.getState(), ARQ::RadioState::IDLE);
    EXPECT_EQ(arq.getTxSuccessCount(), 1u);
}

TEST(Arq, WaitingAckRetryHandlesChannelBusyAndError) {
    NiceMock<MockRadio> busyMock;
    ARQ::ARQ busyArq(busyMock, 1);
    ASSERT_EQ(busyArq.initializeCC1101(), ARQ::STATUS_OK);

    EXPECT_CALL(busyMock, transmitWithCca(_, 47, 2, 5)).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(busyMock, transmitWithCca(_, 47, 2, 3))
        .Times(2)
        .WillRepeatedly(Return(ARQ::RadioStatus::ChannelBusy));
    EXPECT_CALL(busyMock, startReceive(1)).Times(3).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(busyMock, idle()).Times(1);

    uint8_t payload[4] = {1, 2, 3, 4};
    ASSERT_EQ(busyArq.startSendData(payload, 4, 2), ARQ::STATUS_OK);
    EXPECT_EQ(busyArq.poll(0), ARQ::STATUS_BUSY);
    EXPECT_EQ(busyArq.getState(), ARQ::RadioState::WAITING_ACK);
    EXPECT_EQ(busyArq.poll(0), ARQ::STATUS_BUSY);
    EXPECT_EQ(busyArq.getState(), ARQ::RadioState::WAITING_ACK);
    EXPECT_EQ(busyArq.poll(0), ARQ::STATUS_TIMEOUT);
    EXPECT_EQ(busyArq.getTxTimeoutCount(), 1u);
    EXPECT_EQ(busyArq.getTxRetryCount(), 3u);
    EXPECT_EQ(busyArq.getState(), ARQ::RadioState::IDLE);

    NiceMock<MockRadio> errMock;
    ARQ::ARQ errArq(errMock, 1);
    ASSERT_EQ(errArq.initializeCC1101(), ARQ::STATUS_OK);

    EXPECT_CALL(errMock, transmitWithCca(_, 47, 2, 5)).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(errMock, transmitWithCca(_, 47, 2, 3)).WillOnce(Return(ARQ::RadioStatus::Error));
    EXPECT_CALL(errMock, startReceive(1)).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(errMock, idle()).Times(1);

    ASSERT_EQ(errArq.startSendData(payload, 4, 2), ARQ::STATUS_OK);
    EXPECT_EQ(errArq.poll(0), ARQ::STATUS_ERROR);
    EXPECT_EQ(errArq.getState(), ARQ::RadioState::IDLE);
}

TEST(Arq, ReceivingHandlesReadErrorOversizeAndNonDataFrames) {
    NiceMock<MockRadio> mock;
    ARQ::ARQ arq(mock, 2);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    EXPECT_CALL(mock, startReceive(2)).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, readRegField(ARQ::kRegRxBytes, 6, 0)).WillRepeatedly(Return(48));
    ASSERT_EQ(arq.startReceiveData(2), ARQ::STATUS_OK);

    // (a) readData error and short frame < HEADER_SIZE
    EXPECT_CALL(mock, transmit(_, _, _)).Times(0);
    EXPECT_CALL(mock, readData(_, _, _))
        .WillOnce(Return(ARQ::RadioStatus::Error))
        .WillOnce(Invoke([](uint8_t* data, size_t length, size_t* read) {
            (void)data;
            (void)length;
            *read = 6;
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_EQ(arq.poll(), ARQ::STATUS_ERROR);
    EXPECT_FALSE(arq.available());
    EXPECT_EQ(arq.poll(), ARQ::STATUS_ERROR);
    EXPECT_FALSE(arq.available());

    // (b) DATA_FRAME with len = 41 (> 40)
    EXPECT_CALL(mock, readData(_, _, _))
        .WillOnce(Invoke([](uint8_t* data, size_t length, size_t* read) {
            (void)length;
            fillFrame(data, read, 1, ARQ::DATA_FRAME, 1, 41);
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_EQ(arq.poll(), ARQ::STATUS_INVALID_SIZE);
    EXPECT_FALSE(arq.available());

    // (c) DATA_FRAME with len = 40 (exact max payload)
    EXPECT_CALL(mock, readData(_, _, _))
        .WillOnce(Invoke([](uint8_t* data, size_t length, size_t* read) {
            (void)length;
            fillFrame(data, read, 1, ARQ::DATA_FRAME, 1, 40);
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_CALL(mock, transmit(_, 47, 1)).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_EQ(arq.poll(), ARQ::STATUS_OK);
    ASSERT_TRUE(arq.available());
    ARQ::Message msg;
    uint8_t buf[40] = {0};
    ASSERT_EQ(arq.readMessage(msg, buf, sizeof(buf)), ARQ::STATUS_OK);
    EXPECT_EQ(msg.length, 40);

    // (d) valid-version NACK and ACK frames are ignored without ACK reply
    EXPECT_CALL(mock, transmit(_, _, _)).Times(0);
    EXPECT_CALL(mock, readData(_, _, _))
        .WillOnce(Invoke([](uint8_t* data, size_t length, size_t* read) {
            (void)length;
            fillFrame(data, read, 1, ARQ::NACK, 2, 0);
            return ARQ::RadioStatus::Ok;
        }))
        .WillOnce(Invoke([](uint8_t* data, size_t length, size_t* read) {
            (void)length;
            fillFrame(data, read, 1, ARQ::ACK, 3, 0);
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_EQ(arq.poll(), ARQ::STATUS_BUSY);
    EXPECT_FALSE(arq.available());
    EXPECT_EQ(arq.poll(), ARQ::STATUS_BUSY);
    EXPECT_FALSE(arq.available());
}

TEST(Arq, ReceivingDuplicateSameSourceVsDifferentSource) {
    NiceMock<MockRadio> mock;
    ARQ::ARQ arq(mock, 2);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    EXPECT_CALL(mock, startReceive(2)).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, readRegField(ARQ::kRegRxBytes, 6, 0)).WillRepeatedly(Return(48));

    int callIndex = 0;
    EXPECT_CALL(mock, readData(_, _, _))
        .Times(3)
        .WillRepeatedly(Invoke([&](uint8_t* data, size_t length, size_t* read) {
            (void)length;
            ARQ::address_t const src = (callIndex++ < 2) ? 1 : 3;
            fillFrame(data, read, src, ARQ::DATA_FRAME, 5, 4);
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_CALL(mock, transmit(_, 47, 1)).Times(2).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, transmit(_, 47, 3)).Times(1).WillOnce(Return(ARQ::RadioStatus::Ok));

    ASSERT_EQ(arq.startReceiveData(2), ARQ::STATUS_OK);
    EXPECT_EQ(arq.poll(), ARQ::STATUS_OK);
    ASSERT_TRUE(arq.available());
    ARQ::Message msg;
    uint8_t buf[16] = {0};
    ASSERT_EQ(arq.readMessage(msg, buf, sizeof(buf)), ARQ::STATUS_OK);
    EXPECT_EQ(msg.source, 1);

    EXPECT_EQ(arq.poll(), ARQ::STATUS_OK);
    EXPECT_FALSE(arq.available());

    EXPECT_EQ(arq.poll(), ARQ::STATUS_OK);
    ASSERT_TRUE(arq.available());
    ASSERT_EQ(arq.readMessage(msg, buf, sizeof(buf)), ARQ::STATUS_OK);
    EXPECT_EQ(msg.source, 3);
    EXPECT_EQ(msg.sequence, 5);
}

TEST(Arq, ReadMessageEdgeCases) {
    NiceMock<MockRadio> mock;
    ARQ::ARQ arq(mock, 2);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    ARQ::Message msg;
    uint8_t buf[16] = {0};
    EXPECT_EQ(arq.readMessage(msg, buf, sizeof(buf)), ARQ::STATUS_BUSY);

    EXPECT_CALL(mock, startReceive(2)).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, readRegField(ARQ::kRegRxBytes, 6, 0)).WillRepeatedly(Return(48));
    EXPECT_CALL(mock, transmit(_, 47, 1)).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    ASSERT_EQ(arq.startReceiveData(2), ARQ::STATUS_OK);

    // Zero-length DATA_FRAME
    EXPECT_CALL(mock, readData(_, _, _))
        .WillOnce(Invoke([](uint8_t* data, size_t length, size_t* read) {
            (void)length;
            fillFrame(data, read, 1, ARQ::DATA_FRAME, 1, 0);
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_EQ(arq.poll(), ARQ::STATUS_OK);
    ASSERT_TRUE(arq.available());
    EXPECT_EQ(arq.readMessage(msg, buf, sizeof(buf)), ARQ::STATUS_OK);
    EXPECT_EQ(msg.length, 0);
    EXPECT_FALSE(arq.available());

    // nullptr outPayloadBuffer aliases internal rxPayloadBuffer
    EXPECT_CALL(mock, readData(_, _, _))
        .WillOnce(Invoke([](uint8_t* data, size_t length, size_t* read) {
            (void)length;
            fillFrame(data, read, 1, ARQ::DATA_FRAME, 2, 4);
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_EQ(arq.poll(), ARQ::STATUS_OK);
    ASSERT_TRUE(arq.available());
    EXPECT_EQ(arq.readMessage(msg), ARQ::STATUS_OK);
    EXPECT_EQ(msg.length, 4);
    EXPECT_NE(msg.payload, nullptr);
    EXPECT_FALSE(arq.available());

    // Small buffer leaves caller buffer unmodified and aliases internal rxPayloadBuffer
    EXPECT_CALL(mock, readData(_, _, _))
        .WillOnce(Invoke([](uint8_t* data, size_t length, size_t* read) {
            (void)length;
            fillFrame(data, read, 1, ARQ::DATA_FRAME, 3, 4);
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_EQ(arq.poll(), ARQ::STATUS_OK);
    ASSERT_TRUE(arq.available());
    uint8_t small[2] = {0xAA, 0xBB};
    EXPECT_EQ(arq.readMessage(msg, small, sizeof(small)), ARQ::STATUS_OK);
    EXPECT_EQ(small[0], 0xAA);
    EXPECT_EQ(small[1], 0xBB);
    EXPECT_NE(msg.payload, nullptr);
    EXPECT_NE(msg.payload, small);
    EXPECT_FALSE(arq.available());
}

TEST(Arq, DisableCcaUsesTransmitInsteadOfTransmitWithCca) {
    NiceMock<MockRadio> mock;
    ARQ::ARQ arq(mock, 1);
    ASSERT_EQ(arq.initializeCC1101(), ARQ::STATUS_OK);

    arq.enableCca(false);
    EXPECT_FALSE(arq.isCcaEnabled());

    EXPECT_CALL(mock, transmitWithCca(_, _, _, _)).Times(0);
    EXPECT_CALL(mock, transmit(_, 47, 2)).Times(2).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, startReceive(1)).Times(2).WillRepeatedly(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, readRegField(ARQ::kRegRxBytes, 6, 0))
        .WillOnce(Return(0))
        .WillOnce(Return(48));
    EXPECT_CALL(mock, readData(_, _, _))
        .WillOnce(Invoke([](uint8_t* data, size_t length, size_t* read) {
            (void)length;
            fillFrame(data, read, 2, ARQ::ACK, 1, 0);
            return ARQ::RadioStatus::Ok;
        }));
    EXPECT_CALL(mock, idle()).Times(1);

    uint8_t payload[4] = {1, 2, 3, 4};
    ASSERT_EQ(arq.startSendData(payload, 4, 2), ARQ::STATUS_OK);
    EXPECT_EQ(arq.poll(0), ARQ::STATUS_BUSY);
    EXPECT_EQ(arq.poll(1000), ARQ::STATUS_OK);
    EXPECT_EQ(arq.getTxSuccessCount(), 1u);
}
