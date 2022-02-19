package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/selector"
)

type contextKey struct{}

// RequestOptions is a request option.
type RequestOptions struct {
	Filters  []selector.Filter
	Backends []string
}

// NewRequestContext returns a new Context that carries value.
func NewRequestContext(ctx context.Context, o *RequestOptions) context.Context {
	return context.WithValue(ctx, contextKey{}, o)
}

// RequestBackendsFromContext returns backend nodes from context.
func RequestBackendsFromContext(ctx context.Context) ([]string, bool) {
	o, ok := ctx.Value(contextKey{}).(*RequestOptions)
	if ok {
		return o.Backends, true
	}
	return nil, false
}

// WithRequestBackends with backend nodes into context.
func WithRequestBackends(ctx context.Context, backend ...string) context.Context {
	o, ok := ctx.Value(contextKey{}).(*RequestOptions)
	if ok {
		o.Backends = append(o.Backends, backend...)
	}
	return ctx
}

// SelectorFiltersFromContext returns selector filter from context.
func SelectorFiltersFromContext(ctx context.Context) ([]selector.Filter, bool) {
	o, ok := ctx.Value(contextKey{}).(*RequestOptions)
	if ok {
		return o.Filters, true
	}
	return nil, false
}

// WithSelectorFitler with selector filter into context.
func WithSelectorFitler(ctx context.Context, fn selector.Filter) context.Context {
	o, ok := ctx.Value(contextKey{}).(*RequestOptions)
	if ok {
		o.Filters = append(o.Filters, fn)
	}
	return ctx
}
