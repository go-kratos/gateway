package middleware

import (
	"context"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/kratos/v2/selector"
)

type contextKey struct{}

// RequestOptions is a request option.
type RequestOptions struct {
	Endpoint             *config.Endpoint
	Filters              []selector.NodeFilter
	Backends             []string
	Metadata             map[string]string
	UpstreamStatusCode   []int
	UpstreamResponseTime []float64
	DoneFunc             selector.DoneFunc
	LastAttempt          bool
}

// NewRequestOptions new a request options with retry filter.
func NewRequestOptions(c *config.Endpoint) *RequestOptions {
	o := &RequestOptions{
		Endpoint: c,
		Backends: make([]string, 0, 1),
		Metadata: make(map[string]string),
		DoneFunc: func(ctx context.Context, di selector.DoneInfo) {},
	}
	o.Filters = []selector.NodeFilter{func(ctx context.Context, nodes []selector.Node) []selector.Node {
		if len(o.Backends) == 0 {
			return nodes
		}
		selected := make(map[string]struct{}, len(o.Backends))
		for _, b := range o.Backends {
			selected[b] = struct{}{}
		}
		newNodes := nodes[:0]
		for _, node := range nodes {
			if _, ok := selected[node.Address()]; !ok {
				newNodes = append(newNodes, node)
			}
		}
		if len(newNodes) == 0 {
			return nodes
		}
		return newNodes
	}}
	return o
}

// NewRequestContext returns a new Context that carries value.
func NewRequestContext(ctx context.Context, o *RequestOptions) context.Context {
	return context.WithValue(ctx, contextKey{}, o)
}

// FromRequestContext returns request options from context.
func FromRequestContext(ctx context.Context) (*RequestOptions, bool) {
	o, ok := ctx.Value(contextKey{}).(*RequestOptions)
	if ok {
		return o, true
	}
	return nil, false
}

// EndpointFromContext returns endpoint config from context.
func EndpointFromContext(ctx context.Context) (*config.Endpoint, bool) {
	o, ok := ctx.Value(contextKey{}).(*RequestOptions)
	if ok {
		return o.Endpoint, true
	}
	return nil, false
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
func SelectorFiltersFromContext(ctx context.Context) ([]selector.NodeFilter, bool) {
	o, ok := ctx.Value(contextKey{}).(*RequestOptions)
	if ok {
		return o.Filters, true
	}
	return nil, false
}

// WithSelectorFitler with selector filter into context.
func WithSelectorFitler(ctx context.Context, fn selector.NodeFilter) context.Context {
	o, ok := ctx.Value(contextKey{}).(*RequestOptions)
	if ok {
		o.Filters = append(o.Filters, fn)
	}
	return ctx
}
