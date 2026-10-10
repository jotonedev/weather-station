import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from weather.database.v1 import aggregate_pb2 as _aggregate_pb2
from weather.database.v1 import node_pb2 as _node_pb2
from weather.database.v1 import prediction_pb2 as _prediction_pb2
from weather.database.v1 import reading_pb2 as _reading_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class StoreReadingsRequest(_message.Message):
    __slots__ = ("readings",)
    READINGS_FIELD_NUMBER: _ClassVar[int]
    readings: _containers.RepeatedCompositeFieldContainer[_reading_pb2.Reading]
    def __init__(self, readings: _Optional[_Iterable[_Union[_reading_pb2.Reading, _Mapping]]] = ...) -> None: ...

class StoreReadingsResponse(_message.Message):
    __slots__ = ("stored",)
    STORED_FIELD_NUMBER: _ClassVar[int]
    stored: int
    def __init__(self, stored: _Optional[int] = ...) -> None: ...

class StorePredictionsRequest(_message.Message):
    __slots__ = ("predictions",)
    PREDICTIONS_FIELD_NUMBER: _ClassVar[int]
    predictions: _containers.RepeatedCompositeFieldContainer[_prediction_pb2.Prediction]
    def __init__(self, predictions: _Optional[_Iterable[_Union[_prediction_pb2.Prediction, _Mapping]]] = ...) -> None: ...

class StorePredictionsResponse(_message.Message):
    __slots__ = ("stored",)
    STORED_FIELD_NUMBER: _ClassVar[int]
    stored: int
    def __init__(self, stored: _Optional[int] = ...) -> None: ...

class CreateNodeRequest(_message.Message):
    __slots__ = ("node",)
    NODE_FIELD_NUMBER: _ClassVar[int]
    node: _node_pb2.NodeCoordinates
    def __init__(self, node: _Optional[_Union[_node_pb2.NodeCoordinates, _Mapping]] = ...) -> None: ...

class CreateNodeResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class UpdateNodeRequest(_message.Message):
    __slots__ = ("node",)
    NODE_FIELD_NUMBER: _ClassVar[int]
    node: _node_pb2.NodeCoordinates
    def __init__(self, node: _Optional[_Union[_node_pb2.NodeCoordinates, _Mapping]] = ...) -> None: ...

class UpdateNodeResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetReadingsRequest(_message.Message):
    __slots__ = ("node_id", "start_time", "end_time")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    START_TIME_FIELD_NUMBER: _ClassVar[int]
    END_TIME_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    start_time: _timestamp_pb2.Timestamp
    end_time: _timestamp_pb2.Timestamp
    def __init__(self, node_id: _Optional[str] = ..., start_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., end_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetReadingsResponse(_message.Message):
    __slots__ = ("reading",)
    READING_FIELD_NUMBER: _ClassVar[int]
    reading: _reading_pb2.Reading
    def __init__(self, reading: _Optional[_Union[_reading_pb2.Reading, _Mapping]] = ...) -> None: ...

class GetLatestReadingsRequest(_message.Message):
    __slots__ = ("node_id", "limit")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    limit: int
    def __init__(self, node_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class GetLatestReadingsResponse(_message.Message):
    __slots__ = ("readings",)
    READINGS_FIELD_NUMBER: _ClassVar[int]
    readings: _containers.RepeatedCompositeFieldContainer[_reading_pb2.Reading]
    def __init__(self, readings: _Optional[_Iterable[_Union[_reading_pb2.Reading, _Mapping]]] = ...) -> None: ...

class GetForecastRequest(_message.Message):
    __slots__ = ("node_id",)
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    def __init__(self, node_id: _Optional[str] = ...) -> None: ...

class GetForecastResponse(_message.Message):
    __slots__ = ("predictions",)
    PREDICTIONS_FIELD_NUMBER: _ClassVar[int]
    predictions: _containers.RepeatedCompositeFieldContainer[_prediction_pb2.Prediction]
    def __init__(self, predictions: _Optional[_Iterable[_Union[_prediction_pb2.Prediction, _Mapping]]] = ...) -> None: ...

class GetNodeCoordinatesRequest(_message.Message):
    __slots__ = ("node_id",)
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    def __init__(self, node_id: _Optional[str] = ...) -> None: ...

class GetNodeCoordinatesResponse(_message.Message):
    __slots__ = ("node",)
    NODE_FIELD_NUMBER: _ClassVar[int]
    node: _node_pb2.NodeCoordinates
    def __init__(self, node: _Optional[_Union[_node_pb2.NodeCoordinates, _Mapping]] = ...) -> None: ...

class ListNodeCoordinatesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListNodeCoordinatesResponse(_message.Message):
    __slots__ = ("nodes",)
    NODES_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedCompositeFieldContainer[_node_pb2.NodeCoordinates]
    def __init__(self, nodes: _Optional[_Iterable[_Union[_node_pb2.NodeCoordinates, _Mapping]]] = ...) -> None: ...

class GetClimateAggregatesRequest(_message.Message):
    __slots__ = ("node_id", "start_time", "end_time", "bucket_seconds")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    START_TIME_FIELD_NUMBER: _ClassVar[int]
    END_TIME_FIELD_NUMBER: _ClassVar[int]
    BUCKET_SECONDS_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    start_time: _timestamp_pb2.Timestamp
    end_time: _timestamp_pb2.Timestamp
    bucket_seconds: int
    def __init__(self, node_id: _Optional[str] = ..., start_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., end_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., bucket_seconds: _Optional[int] = ...) -> None: ...

class GetClimateAggregatesResponse(_message.Message):
    __slots__ = ("aggregates",)
    AGGREGATES_FIELD_NUMBER: _ClassVar[int]
    aggregates: _containers.RepeatedCompositeFieldContainer[_aggregate_pb2.ClimateAggregate]
    def __init__(self, aggregates: _Optional[_Iterable[_Union[_aggregate_pb2.ClimateAggregate, _Mapping]]] = ...) -> None: ...

class GetAirQualityAggregatesRequest(_message.Message):
    __slots__ = ("node_id", "start_time", "end_time", "bucket_seconds")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    START_TIME_FIELD_NUMBER: _ClassVar[int]
    END_TIME_FIELD_NUMBER: _ClassVar[int]
    BUCKET_SECONDS_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    start_time: _timestamp_pb2.Timestamp
    end_time: _timestamp_pb2.Timestamp
    bucket_seconds: int
    def __init__(self, node_id: _Optional[str] = ..., start_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., end_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., bucket_seconds: _Optional[int] = ...) -> None: ...

class GetAirQualityAggregatesResponse(_message.Message):
    __slots__ = ("aggregates",)
    AGGREGATES_FIELD_NUMBER: _ClassVar[int]
    aggregates: _containers.RepeatedCompositeFieldContainer[_aggregate_pb2.AirQualityAggregate]
    def __init__(self, aggregates: _Optional[_Iterable[_Union[_aggregate_pb2.AirQualityAggregate, _Mapping]]] = ...) -> None: ...
