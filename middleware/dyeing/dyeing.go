package dyeing

import (
	"context"
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/dyeing/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/selector"
)

// Name is the middleware name.
const Name = "dyeing"

func init() {
	middleware.Register(Name, Middleware)
}

// Middleware .
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Dyeing{}
	if err := c.Options.UnmarshalTo(options); err != nil {
		return nil, err
	}
	return func(handler middleware.Endpoint) middleware.Endpoint {
		return func(ctx context.Context, req *http.Request) (reply *http.Response, err error) {
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
				if options, ok := middleware.FromContext(ctx); ok {
					options.Filters = append(options.Filters, filter)
				}
			}
			return handler(ctx, req)
		}
	}, nil
}
