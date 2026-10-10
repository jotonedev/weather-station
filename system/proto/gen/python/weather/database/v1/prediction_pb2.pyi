import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Prediction(_message.Message):
    __slots__ = ("node_id", "issue_time", "horizon_hours", "rainfall_mm", "temperature_c", "humidity_pct", "wind_direction_deg", "rainfall_confidence", "temperature_confidence", "humidity_confidence", "wind_direction_confidence")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ISSUE_TIME_FIELD_NUMBER: _ClassVar[int]
    HORIZON_HOURS_FIELD_NUMBER: _ClassVar[int]
    RAINFALL_MM_FIELD_NUMBER: _ClassVar[int]
    TEMPERATURE_C_FIELD_NUMBER: _ClassVar[int]
    HUMIDITY_PCT_FIELD_NUMBER: _ClassVar[int]
    WIND_DIRECTION_DEG_FIELD_NUMBER: _ClassVar[int]
    RAINFALL_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    TEMPERATURE_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    HUMIDITY_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    WIND_DIRECTION_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    issue_time: _timestamp_pb2.Timestamp
    horizon_hours: int
    rainfall_mm: float
    temperature_c: float
    humidity_pct: float
    wind_direction_deg: float
    rainfall_confidence: float
    temperature_confidence: float
    humidity_confidence: float
    wind_direction_confidence: float
    def __init__(self, node_id: _Optional[str] = ..., issue_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., horizon_hours: _Optional[int] = ..., rainfall_mm: _Optional[float] = ..., temperature_c: _Optional[float] = ..., humidity_pct: _Optional[float] = ..., wind_direction_deg: _Optional[float] = ..., rainfall_confidence: _Optional[float] = ..., temperature_confidence: _Optional[float] = ..., humidity_confidence: _Optional[float] = ..., wind_direction_confidence: _Optional[float] = ...) -> None: ...
