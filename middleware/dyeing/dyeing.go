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
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			color := req.Header.Get(options.Header)
			if color != "" {
				f := func(_ context.Context, nodes []selector.Node) []selector.Node {
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
					options.Filters = append(options.Filters, f)
				}
			}
			next.ServeHTTP(w, req)
		})
	}, nil
}
