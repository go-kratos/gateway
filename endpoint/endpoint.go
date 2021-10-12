package endpoint

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

// Request is an HTTP request.
type Request interface {
	Path() string
	Method() string
	Query() url.Values
	Header() http.Header
	Body() io.ReadCloser
}

// Response is an HTTP response.
type Response interface {
	Header() http.Header
	Trailer() http.Header
	StatusCode() int
	Body() io.ReadCloser
}

// Endpoint defines the endpoint invoked by Middleware.
type Endpoint func(context.Context, Request) (Response, error)

// Middleware is endpoint middleware.
type Middleware func(Endpoint) Endpoint
