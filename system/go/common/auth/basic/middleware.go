package basic

import (
	"crypto/subtle"
	"net/http"
)

// Authenticator validates HTTP Basic Authentication credentials in constant time.
type Authenticator struct {
	username []byte
	password []byte
}

// NewAuthenticator creates a new Authenticator from the provided configuration.
func NewAuthenticator(cfg Config) *Authenticator {
	return &Authenticator{
		username: []byte(cfg.Username),
		password: []byte(cfg.Password),
	}
}

// Check reports whether the request carries valid Basic Auth credentials.
func (a *Authenticator) Check(r *http.Request) bool {
	if a == nil || len(a.username) == 0 || len(a.password) == 0 {
		return false
	}

	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}

	userMatch := subtle.ConstantTimeCompare([]byte(user), a.username) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(pass), a.password) == 1
	return userMatch && passMatch
}

// RequireAuth validates request credentials and writes a 401 response when invalid.
func (a *Authenticator) RequireAuth(w http.ResponseWriter, r *http.Request) bool {
	if a.Check(r) {
		return true
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Weather Station"`)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
	return false
}
