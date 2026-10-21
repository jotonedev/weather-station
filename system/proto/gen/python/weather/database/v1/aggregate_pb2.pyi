import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ClimateAggregate(_message.Message):
    __slots__ = ("node_id", "bucket_start", "avg_temperature_c", "min_temperature_c", "max_temperature_c", "avg_humidity_pct", "min_humidity_pct", "max_humidity_pct", "avg_pressure_hpa", "sample_count", "total_rainfall_mm", "avg_wind_speed_mps", "avg_wind_direction_deg")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    BUCKET_START_FIELD_NUMBER: _ClassVar[int]
    AVG_TEMPERATURE_C_FIELD_NUMBER: _ClassVar[int]
    MIN_TEMPERATURE_C_FIELD_NUMBER: _ClassVar[int]
    MAX_TEMPERATURE_C_FIELD_NUMBER: _ClassVar[int]
    AVG_HUMIDITY_PCT_FIELD_NUMBER: _ClassVar[int]
    MIN_HUMIDITY_PCT_FIELD_NUMBER: _ClassVar[int]
    MAX_HUMIDITY_PCT_FIELD_NUMBER: _ClassVar[int]
    AVG_PRESSURE_HPA_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_RAINFALL_MM_FIELD_NUMBER: _ClassVar[int]
    AVG_WIND_SPEED_MPS_FIELD_NUMBER: _ClassVar[int]
    AVG_WIND_DIRECTION_DEG_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    bucket_start: _timestamp_pb2.Timestamp
    avg_temperature_c: float
    min_temperature_c: float
    max_temperature_c: float
    avg_humidity_pct: float
    min_humidity_pct: float
    max_humidity_pct: float
    avg_pressure_hpa: float
    sample_count: int
    total_rainfall_mm: float
    avg_wind_speed_mps: float
    avg_wind_direction_deg: float
    def __init__(self, node_id: _Optional[str] = ..., bucket_start: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., avg_temperature_c: _Optional[float] = ..., min_temperature_c: _Optional[float] = ..., max_temperature_c: _Optional[float] = ..., avg_humidity_pct: _Optional[float] = ..., min_humidity_pct: _Optional[float] = ..., max_humidity_pct: _Optional[float] = ..., avg_pressure_hpa: _Optional[float] = ..., sample_count: _Optional[int] = ..., total_rainfall_mm: _Optional[float] = ..., avg_wind_speed_mps: _Optional[float] = ..., avg_wind_direction_deg: _Optional[float] = ...) -> None: ...

class AirQualityAggregate(_message.Message):
    __slots__ = ("node_id", "bucket_start", "avg_pm1", "avg_pm25", "avg_pm10", "avg_voc", "sample_count")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    BUCKET_START_FIELD_NUMBER: _ClassVar[int]
    AVG_PM1_FIELD_NUMBER: _ClassVar[int]
    AVG_PM25_FIELD_NUMBER: _ClassVar[int]
    AVG_PM10_FIELD_NUMBER: _ClassVar[int]
    AVG_VOC_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    bucket_start: _timestamp_pb2.Timestamp
    avg_pm1: float
    avg_pm25: float
    avg_pm10: float
    avg_voc: float
    sample_count: int
    def __init__(self, node_id: _Optional[str] = ..., bucket_start: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., avg_pm1: _Optional[float] = ..., avg_pm25: _Optional[float] = ..., avg_pm10: _Optional[float] = ..., avg_voc: _Optional[float] = ..., sample_count: _Optional[int] = ...) -> None: ...
