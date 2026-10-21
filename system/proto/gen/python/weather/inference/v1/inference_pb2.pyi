import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from weather.database.v1 import prediction_pb2 as _prediction_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PredictHourlyWeatherRequest(_message.Message):
    __slots__ = ("node_id", "issue_time", "horizon_hours", "history", "latitude", "data_blob", "data_blob_format")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ISSUE_TIME_FIELD_NUMBER: _ClassVar[int]
    HORIZON_HOURS_FIELD_NUMBER: _ClassVar[int]
    HISTORY_FIELD_NUMBER: _ClassVar[int]
    LATITUDE_FIELD_NUMBER: _ClassVar[int]
    DATA_BLOB_FIELD_NUMBER: _ClassVar[int]
    DATA_BLOB_FORMAT_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    issue_time: _timestamp_pb2.Timestamp
    horizon_hours: int
    history: _containers.RepeatedCompositeFieldContainer[TimeSeriesObservation]
    latitude: float
    data_blob: bytes
    data_blob_format: str
    def __init__(self, node_id: _Optional[str] = ..., issue_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., horizon_hours: _Optional[int] = ..., history: _Optional[_Iterable[_Union[TimeSeriesObservation, _Mapping]]] = ..., latitude: _Optional[float] = ..., data_blob: _Optional[bytes] = ..., data_blob_format: _Optional[str] = ...) -> None: ...

class PredictHourlyWeatherResponse(_message.Message):
    __slots__ = ("predictions", "computed_at", "output_blob")
    PREDICTIONS_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_BLOB_FIELD_NUMBER: _ClassVar[int]
    predictions: _containers.RepeatedCompositeFieldContainer[_prediction_pb2.HourlyPrediction]
    computed_at: _timestamp_pb2.Timestamp
    output_blob: bytes
    def __init__(self, predictions: _Optional[_Iterable[_Union[_prediction_pb2.HourlyPrediction, _Mapping]]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., output_blob: _Optional[bytes] = ...) -> None: ...

class PredictDailyWeatherRequest(_message.Message):
    __slots__ = ("node_id", "issue_time", "horizon_days", "history", "latitude", "data_blob", "data_blob_format")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ISSUE_TIME_FIELD_NUMBER: _ClassVar[int]
    HORIZON_DAYS_FIELD_NUMBER: _ClassVar[int]
    HISTORY_FIELD_NUMBER: _ClassVar[int]
    LATITUDE_FIELD_NUMBER: _ClassVar[int]
    DATA_BLOB_FIELD_NUMBER: _ClassVar[int]
    DATA_BLOB_FORMAT_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    issue_time: _timestamp_pb2.Timestamp
    horizon_days: int
    history: _containers.RepeatedCompositeFieldContainer[TimeSeriesObservation]
    latitude: float
    data_blob: bytes
    data_blob_format: str
    def __init__(self, node_id: _Optional[str] = ..., issue_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., horizon_days: _Optional[int] = ..., history: _Optional[_Iterable[_Union[TimeSeriesObservation, _Mapping]]] = ..., latitude: _Optional[float] = ..., data_blob: _Optional[bytes] = ..., data_blob_format: _Optional[str] = ...) -> None: ...

class PredictDailyWeatherResponse(_message.Message):
    __slots__ = ("predictions", "computed_at", "output_blob")
    PREDICTIONS_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_BLOB_FIELD_NUMBER: _ClassVar[int]
    predictions: _containers.RepeatedCompositeFieldContainer[_prediction_pb2.DailyPrediction]
    computed_at: _timestamp_pb2.Timestamp
    output_blob: bytes
    def __init__(self, predictions: _Optional[_Iterable[_Union[_prediction_pb2.DailyPrediction, _Mapping]]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., output_blob: _Optional[bytes] = ...) -> None: ...

class TimeSeriesObservation(_message.Message):
    __slots__ = ("timestamp", "temperature_c", "humidity_pct", "pressure_hpa", "wind_speed_mps", "wind_direction_deg", "rainfall_mm", "min_temperature_c", "max_temperature_c", "min_humidity_pct", "max_humidity_pct", "min_pressure_hpa", "max_pressure_hpa")
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    TEMPERATURE_C_FIELD_NUMBER: _ClassVar[int]
    HUMIDITY_PCT_FIELD_NUMBER: _ClassVar[int]
    PRESSURE_HPA_FIELD_NUMBER: _ClassVar[int]
    WIND_SPEED_MPS_FIELD_NUMBER: _ClassVar[int]
    WIND_DIRECTION_DEG_FIELD_NUMBER: _ClassVar[int]
    RAINFALL_MM_FIELD_NUMBER: _ClassVar[int]
    MIN_TEMPERATURE_C_FIELD_NUMBER: _ClassVar[int]
    MAX_TEMPERATURE_C_FIELD_NUMBER: _ClassVar[int]
    MIN_HUMIDITY_PCT_FIELD_NUMBER: _ClassVar[int]
    MAX_HUMIDITY_PCT_FIELD_NUMBER: _ClassVar[int]
    MIN_PRESSURE_HPA_FIELD_NUMBER: _ClassVar[int]
    MAX_PRESSURE_HPA_FIELD_NUMBER: _ClassVar[int]
    timestamp: _timestamp_pb2.Timestamp
    temperature_c: float
    humidity_pct: float
    pressure_hpa: float
    wind_speed_mps: float
    wind_direction_deg: float
    rainfall_mm: float
    min_temperature_c: float
    max_temperature_c: float
    min_humidity_pct: float
    max_humidity_pct: float
    min_pressure_hpa: float
    max_pressure_hpa: float
    def __init__(self, timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., temperature_c: _Optional[float] = ..., humidity_pct: _Optional[float] = ..., pressure_hpa: _Optional[float] = ..., wind_speed_mps: _Optional[float] = ..., wind_direction_deg: _Optional[float] = ..., rainfall_mm: _Optional[float] = ..., min_temperature_c: _Optional[float] = ..., max_temperature_c: _Optional[float] = ..., min_humidity_pct: _Optional[float] = ..., max_humidity_pct: _Optional[float] = ..., min_pressure_hpa: _Optional[float] = ..., max_pressure_hpa: _Optional[float] = ...) -> None: ...

class GetModelInfoRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetModelInfoResponse(_message.Message):
    __slots__ = ("model_name", "max_horizon_hours")
    MODEL_NAME_FIELD_NUMBER: _ClassVar[int]
    MAX_HORIZON_HOURS_FIELD_NUMBER: _ClassVar[int]
    model_name: str
    max_horizon_hours: int
    def __init__(self, model_name: _Optional[str] = ..., max_horizon_hours: _Optional[int] = ...) -> None: ...
