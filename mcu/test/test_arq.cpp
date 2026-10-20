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

// Fill a header plus 4-byte payload into the caller buffer, as a received frame.
void fillDataFrame(uint8_t* data, size_t length, size_t* read, ARQ::seq_t seq) {
    (void)length;
    data[0] = ARQ::PROTOCOL_VERSION;
    data[1] = 1; // source is the control node
    data[2] = ARQ::DATA_FRAME;
    data[3] = static_cast<uint8_t>(seq & 0xFF);
    data[4] = static_cast<uint8_t>((seq >> 8) & 0xFF);
    data[5] = 4; // length
    data[6] = 0;
    data[7] = 0x01;
    data[8] = 0x02;
    data[9] = 0x03;
    data[10] = 0x04;
    *read = 11;
}

} // namespace

TEST(Arq, InitializesRadioConfiguration) {
    NiceMock<MockRadio> mock;
    EXPECT_CALL(mock, begin()).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, setModulation(ARQ::Modulation::Gfsk));
    EXPECT_CALL(mock, setFrequency(433.5)).WillOnce(Return(ARQ::RadioStatus::Ok));
    EXPECT_CALL(mock, setPacketLengthMode(ARQ::PacketLengthMode::Fixed, 60));
    EXPECT_CALL(mock, setReceiveAction(_, ARQ::GdoPin::Gdo0));
    EXPECT_CALL(mock, writeReg(ARQ::kRegIocfg0, 0x07));

    ARQ::ARQ arq(mock, 1);
    EXPECT_EQ(arq.initializeCC1101(60), ARQ::STATUS_OK);
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
