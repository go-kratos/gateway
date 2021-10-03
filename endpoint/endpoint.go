package endpoint

import "net/http"

// Endpoint is an HTTP handler.
type Endpoint http.Handler

// Middleware is a function which receives an http.Handler and returns another http.Handler.
type Middleware func(http.Handler) http.Handler
