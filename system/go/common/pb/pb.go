// Package pb holds helpers for converting between Go and protobuf types.
package pb

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Time converts a Go time into a protobuf timestamp.
func Time(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

// FromTime converts a protobuf timestamp into a Go time.
func FromTime(ts *timestamppb.Timestamp) time.Time {
	return ts.AsTime()
}
