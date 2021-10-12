package dyeing

import (
	"context"
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/dyeing/v1"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/kratos/v2/selector"

	"github.com/go-kratos/gateway/middleware"
)

// Name is the middleware name.
const Name = "dyeing"

// Middleware .
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Dyeing{}
	if err := c.Options.UnmarshalTo(options); err != nil {
		return nil, err
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (reply middleware.Response, err error) {
			if color := req.Header.Get(options.Header); color != "" {
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
				if options, ok := proxy.FromContext(req.Context()); ok {
					options.Filters = append(options.Filters, filter)
				}
			}

			return handler(ctx, req)
		}
	}, nil
}
