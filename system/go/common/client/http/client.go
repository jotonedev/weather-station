package http

import (
	"net/http"
)

func NewHTTPClient() *http.Client {
	client := &http.Client{}

	return client
}
