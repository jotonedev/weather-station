from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class NodeCoordinates(_message.Message):
    __slots__ = ("node_id", "latitude", "longitude", "elevation_m", "location_name")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    LATITUDE_FIELD_NUMBER: _ClassVar[int]
    LONGITUDE_FIELD_NUMBER: _ClassVar[int]
    ELEVATION_M_FIELD_NUMBER: _ClassVar[int]
    LOCATION_NAME_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    latitude: float
    longitude: float
    elevation_m: float
    location_name: str
    def __init__(self, node_id: _Optional[str] = ..., latitude: _Optional[float] = ..., longitude: _Optional[float] = ..., elevation_m: _Optional[float] = ..., location_name: _Optional[str] = ...) -> None: ...
