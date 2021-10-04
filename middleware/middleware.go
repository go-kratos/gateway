package middleware

import "net/http"

// Middleware is a function which receives an http.Handler and returns another http.Handler.
type Middleware func(http.Handler) http.Handler
