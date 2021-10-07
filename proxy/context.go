package proxy

import "context"

type contextKey struct{}

// RequestOptions is a request option.
type RequestOptions struct {
	Labels map[string]string
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
