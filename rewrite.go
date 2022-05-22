package rewrite

import (
	"net/http"
	"strings"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/rewrite/v1"

	"github.com/go-kratos/gateway/middleware"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func init() {
	middleware.Register("rewrite", Middleware)
}

func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.ReplacePath{}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}
	pathBuilder := func(req *http.Request) string {
		return options.DestPath
	}
	if len(options.PathParam) > 0 {
		pathBuilder = func(req *http.Request) string {
			out := options.DestPath
			for key, keySchema := range options.PathParam {
				switch keySchema.Type {
				case v1.KeySchemaType_QUERY:
					query := req.URL.Query()
					out = strings.ReplaceAll(options.DestPath, "{"+key+"}", query.Get(keySchema.Key))
					if keySchema.Strip {
						query.Del(keySchema.Key)
						req.URL.RawQuery = query.Encode()
					}
				}
			}
			return out
		}
	}

	header := &v1.Header{}
	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			dstPath := pathBuilder(req)
			req.URL.Path = dstPath
			for key, value := range header.HttpHeaders {
				req.Header.Set(key, value)
			}
			return next.RoundTrip(req)
		})
	}, nil
}
