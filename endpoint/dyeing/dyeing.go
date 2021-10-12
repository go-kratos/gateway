package dyeing

import (
	"context"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/dyeing/v1"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/kratos/v2/selector"

	"github.com/go-kratos/gateway/endpoint"
)

// Name is the middleware name.
const Name = "dyeing"

// Middleware .
func Middleware(c *config.Middleware) (endpoint.Middleware, error) {
	options := &v1.Dyeing{}
	if err := c.Options.UnmarshalTo(options); err != nil {
		return nil, err
	}
	return func(handler endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req endpoint.Request) (reply endpoint.Response, err error) {
			if color := req.Header().Get(options.Header); color != "" {
				filter := func(ctx context.Context, nodes []selector.Node) []selector.Node {
					filtered := make([]selector.Node, 0, len(nodes))
					for _, n := range nodes {
						md := n.Metadata()
						if md[options.Label] == color {
							filtered = append(filtered, n)
						}
					}
					if len(filtered) == 0 {
						for _, n := range nodes {
							md := n.Metadata()
							if _, ok := md[options.Label]; !ok {
								filtered = append(filtered, n)
							}
						}
					}
					return filtered
				}
				if options, ok := proxy.FromContext(ctx); ok {
					options.Filters = append(options.Filters, filter)
				}
			}
			return handler(ctx, req)
		}
	}, nil
}
