package middleware

import (
	"context"
	"net/http"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
)

// Handler defines the handler invoked by Middleware.
type Handler func(context.Context, *http.Request) (*http.Response, error)

// Middleware is handler middleware.
type Middleware func(Handler) Handler

// Factory is a middleware factory.
type Factory func(*configv1.Middleware) (Middleware, error)
