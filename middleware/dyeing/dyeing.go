package dyeing

import (
	"context"
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/kratos/v2/selector"

	"github.com/go-kratos/gateway/middleware"
)

// Name is the middleware name.
const Name = "dyeing"

// Middleware .
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	colorLabel := "color"
	if c.Options != nil {
		if v := c.Options.Fields["label"]; v != nil {
			colorLabel = v.GetStringValue()
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			color := req.Header.Get(colorLabel)
			if color != "" {
				f := func(_ context.Context, nodes []selector.Node) []selector.Node {
					filtered := make([]selector.Node, 0, len(nodes))
					for _, n := range nodes {
						md := n.Metadata()
						if md[colorLabel] == color {
							filtered = append(filtered, n)
						}
					}
					if len(filtered) == 0 {
						for _, n := range nodes {
							md := n.Metadata()
							if _, ok := md[colorLabel]; !ok {
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
