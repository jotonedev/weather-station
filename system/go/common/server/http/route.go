package http

import "net/http"

type Route interface {
	// Handler returns the HTTP handler for this route.
	http.Handler

	// Pattern reports the path at which this is registered.
	Pattern() string
}
