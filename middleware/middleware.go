package middleware

import (
	"net/http"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
)

// Middleware is handler middleware.
type Middleware func(http.RoundTripper) http.RoundTripper

// Factory is a middleware factory.
type Factory func(*configv1.Middleware) (Middleware, error)

// RoundTripperFunc is an adapter to allow the use of
// ordinary functions as HTTP RoundTripper.
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls f(w, r).
func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
