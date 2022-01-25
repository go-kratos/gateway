package color

import (
	"context"
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/color/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/selector"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func init() {
	middleware.Register("color", Middleware)
}

func filter(label, color string) func(ctx context.Context, nodes []selector.Node) []selector.Node {
	return func(ctx context.Context, nodes []selector.Node) []selector.Node {
		filtered := make([]selector.Node, 0, len(nodes))
		for _, n := range nodes {
			md := n.Metadata()
			if md[label] == color {
				filtered = append(filtered, n)
			}
		}
		if len(filtered) == 0 {
			for _, n := range nodes {
				md := n.Metadata()
				if _, ok := md[label]; !ok {
					filtered = append(filtered, n)
				}
			}
		}
		return filtered
	}
}

// Middleware is a dyeing request to filter the color nodes.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Color{
		Header: "x-md-global-color",
		Label:  "color",
	}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (reply *http.Response, err error) {
			if color := req.Header.Get(options.Header); color != "" {
				if o, ok := middleware.FromRequestContext(ctx); ok {
					o.Filters = append(o.Filters, filter(options.Label, color))
				}
			}
			return handler(ctx, req)
		}
	}, nil
}
