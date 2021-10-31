package endpoint

import (
	"context"
	"net/http"
)

// Endpoint defines the endpoint invoked by Middleware.
type Endpoint func(context.Context, *http.Request) (*http.Response, error)

// Middleware is endpoint middleware.
type Middleware func(Endpoint) Endpoint
