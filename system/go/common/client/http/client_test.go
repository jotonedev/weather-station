package http

import (
	"testing"
)

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Error("Expected non-nil client")
	}
}
