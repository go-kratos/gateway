package middleware

import (
	"io"
	"net/http"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
)

// Factory is a middleware factory.
type Factory func(*configv1.Middleware) (Middleware, error)

// Middleware is handler middleware.
type Middleware func(http.RoundTripper) http.RoundTripper

// RoundTripperFunc is an adapter to allow the use of
// ordinary functions as HTTP RoundTripper.
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls f(w, r).
func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type FactoryV2 func(*configv1.Middleware) (MiddlewareV2, error)
type MiddlewareV2 interface {
	Process(http.RoundTripper) http.RoundTripper
	io.Closer
}

func wrapFactory(in Factory) FactoryV2 {
	return func(m *configv1.Middleware) (MiddlewareV2, error) {
		v, err := in(m)
		if err != nil {
			return nil, err
		}
		return v, nil
	}
}

func (f Middleware) Process(in http.RoundTripper) http.RoundTripper { return f(in) }
func (f Middleware) Close() error                                   { return nil }

type withCloser struct {
	process Middleware
	closer  io.Closer
}

func (w *withCloser) Process(in http.RoundTripper) http.RoundTripper { return w.process(in) }
func (w *withCloser) Close() error                                   { return w.closer.Close() }
func NewWithCloser(process Middleware, closer io.Closer) MiddlewareV2 {
	return &withCloser{
		process: process,
		closer:  closer,
	}
}
