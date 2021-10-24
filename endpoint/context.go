package endpoint

import (
	"context"

	"github.com/go-kratos/kratos/v2/selector"
)

type contextKey struct{}

// RequestOptions is a request option.
type RequestOptions struct {
	Filters   []selector.Filter
	UsedNodes []selector.Node
}

// NewContext returns a new Context that carries value.
func NewContext(ctx context.Context, o *RequestOptions) context.Context {
	return context.WithValue(ctx, contextKey{}, o)
}

// FromContext returns the Context value stored in ctx, if any.
func FromContext(ctx context.Context) (o *RequestOptions, ok bool) {
	o, ok = ctx.Value(contextKey{}).(*RequestOptions)
	return
}
