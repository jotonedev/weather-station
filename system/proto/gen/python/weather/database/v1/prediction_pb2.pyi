import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class HourlyPrediction(_message.Message):
    __slots__ = ("node_id", "issue_time", "horizon_hours", "temperature_c", "humidity_pct", "rainfall_mm", "rainfall_confidence")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ISSUE_TIME_FIELD_NUMBER: _ClassVar[int]
    HORIZON_HOURS_FIELD_NUMBER: _ClassVar[int]
    TEMPERATURE_C_FIELD_NUMBER: _ClassVar[int]
    HUMIDITY_PCT_FIELD_NUMBER: _ClassVar[int]
    RAINFALL_MM_FIELD_NUMBER: _ClassVar[int]
    RAINFALL_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    issue_time: _timestamp_pb2.Timestamp
    horizon_hours: int
    temperature_c: float
    humidity_pct: float
    rainfall_mm: float
    rainfall_confidence: float
    def __init__(self, node_id: _Optional[str] = ..., issue_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., horizon_hours: _Optional[int] = ..., temperature_c: _Optional[float] = ..., humidity_pct: _Optional[float] = ..., rainfall_mm: _Optional[float] = ..., rainfall_confidence: _Optional[float] = ...) -> None: ...

class DailyPrediction(_message.Message):
    __slots__ = ("node_id", "issue_time", "horizon_days", "min_temperature_c", "max_temperature_c", "min_humidity_pct", "max_humidity_pct", "rainfall_mm", "rainfall_confidence")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ISSUE_TIME_FIELD_NUMBER: _ClassVar[int]
    HORIZON_DAYS_FIELD_NUMBER: _ClassVar[int]
    MIN_TEMPERATURE_C_FIELD_NUMBER: _ClassVar[int]
    MAX_TEMPERATURE_C_FIELD_NUMBER: _ClassVar[int]
    MIN_HUMIDITY_PCT_FIELD_NUMBER: _ClassVar[int]
    MAX_HUMIDITY_PCT_FIELD_NUMBER: _ClassVar[int]
    RAINFALL_MM_FIELD_NUMBER: _ClassVar[int]
    RAINFALL_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    issue_time: _timestamp_pb2.Timestamp
    horizon_days: int
    min_temperature_c: float
    max_temperature_c: float
    min_humidity_pct: float
    max_humidity_pct: float
    rainfall_mm: float
    rainfall_confidence: float
    def __init__(self, node_id: _Optional[str] = ..., issue_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., horizon_days: _Optional[int] = ..., min_temperature_c: _Optional[float] = ..., max_temperature_c: _Optional[float] = ..., min_humidity_pct: _Optional[float] = ..., max_humidity_pct: _Optional[float] = ..., rainfall_mm: _Optional[float] = ..., rainfall_confidence: _Optional[float] = ...) -> None: ...
