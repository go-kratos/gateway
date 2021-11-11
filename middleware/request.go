package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/selector"
)

type contextKey struct{}

// RequestOptions is a request option.
type RequestOptions struct {
	Filters []selector.NodeFilter
}

// NewRequestContext returns a new Context that carries value.
func NewRequestContext(ctx context.Context, o *RequestOptions) context.Context {
	return context.WithValue(ctx, contextKey{}, o)
}

// FromRequestContext returns the Context value stored in ctx, if any.
func FromRequestContext(ctx context.Context) (o *RequestOptions, ok bool) {
	o, ok = ctx.Value(contextKey{}).(*RequestOptions)
	return
}
