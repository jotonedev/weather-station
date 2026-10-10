import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Reading(_message.Message):
    __slots__ = ("node_id", "timestamp", "wind_speed", "wind_angle", "raw_wind_speed", "raw_wind_angle", "temperature", "humidity", "pressure", "rain1h", "rain24h", "rain_rate", "pm1", "pm25", "pm10", "voc", "anemo_status", "anemo_agc", "anemo_agc_variance", "vane_status", "vane_agc", "vane_agc_variance", "rain_status", "rain_agc", "rain_agc_variance")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    WIND_SPEED_FIELD_NUMBER: _ClassVar[int]
    WIND_ANGLE_FIELD_NUMBER: _ClassVar[int]
    RAW_WIND_SPEED_FIELD_NUMBER: _ClassVar[int]
    RAW_WIND_ANGLE_FIELD_NUMBER: _ClassVar[int]
    TEMPERATURE_FIELD_NUMBER: _ClassVar[int]
    HUMIDITY_FIELD_NUMBER: _ClassVar[int]
    PRESSURE_FIELD_NUMBER: _ClassVar[int]
    RAIN1H_FIELD_NUMBER: _ClassVar[int]
    RAIN24H_FIELD_NUMBER: _ClassVar[int]
    RAIN_RATE_FIELD_NUMBER: _ClassVar[int]
    PM1_FIELD_NUMBER: _ClassVar[int]
    PM25_FIELD_NUMBER: _ClassVar[int]
    PM10_FIELD_NUMBER: _ClassVar[int]
    VOC_FIELD_NUMBER: _ClassVar[int]
    ANEMO_STATUS_FIELD_NUMBER: _ClassVar[int]
    ANEMO_AGC_FIELD_NUMBER: _ClassVar[int]
    ANEMO_AGC_VARIANCE_FIELD_NUMBER: _ClassVar[int]
    VANE_STATUS_FIELD_NUMBER: _ClassVar[int]
    VANE_AGC_FIELD_NUMBER: _ClassVar[int]
    VANE_AGC_VARIANCE_FIELD_NUMBER: _ClassVar[int]
    RAIN_STATUS_FIELD_NUMBER: _ClassVar[int]
    RAIN_AGC_FIELD_NUMBER: _ClassVar[int]
    RAIN_AGC_VARIANCE_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    timestamp: _timestamp_pb2.Timestamp
    wind_speed: float
    wind_angle: float
    raw_wind_speed: float
    raw_wind_angle: float
    temperature: float
    humidity: float
    pressure: float
    rain1h: float
    rain24h: float
    rain_rate: float
    pm1: int
    pm25: int
    pm10: int
    voc: int
    anemo_status: int
    anemo_agc: int
    anemo_agc_variance: int
    vane_status: int
    vane_agc: int
    vane_agc_variance: int
    rain_status: int
    rain_agc: int
    rain_agc_variance: int
    def __init__(self, node_id: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., wind_speed: _Optional[float] = ..., wind_angle: _Optional[float] = ..., raw_wind_speed: _Optional[float] = ..., raw_wind_angle: _Optional[float] = ..., temperature: _Optional[float] = ..., humidity: _Optional[float] = ..., pressure: _Optional[float] = ..., rain1h: _Optional[float] = ..., rain24h: _Optional[float] = ..., rain_rate: _Optional[float] = ..., pm1: _Optional[int] = ..., pm25: _Optional[int] = ..., pm10: _Optional[int] = ..., voc: _Optional[int] = ..., anemo_status: _Optional[int] = ..., anemo_agc: _Optional[int] = ..., anemo_agc_variance: _Optional[int] = ..., vane_status: _Optional[int] = ..., vane_agc: _Optional[int] = ..., vane_agc_variance: _Optional[int] = ..., rain_status: _Optional[int] = ..., rain_agc: _Optional[int] = ..., rain_agc_variance: _Optional[int] = ...) -> None: ...
