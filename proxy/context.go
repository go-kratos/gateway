package proxy

import "context"

type contextKey struct{}

// Context is a proxy context.
type Context struct {
	Labels map[string]string
}

// NewContext returns a new Context that carries value.
func NewContext(ctx context.Context, c *Context) context.Context {
	return context.WithValue(ctx, contextKey{}, c)
}

// FromContext returns the Context value stored in ctx, if any.
func FromContext(ctx context.Context) (c *Context, ok bool) {
	c, ok = ctx.Value(contextKey{}).(*Context)
	return
}
