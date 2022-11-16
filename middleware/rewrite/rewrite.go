package rewrite

import (
	"net/http"
	"path"
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

func stripPrefix(origin string, prefix string) string {
	out := strings.TrimPrefix(origin, prefix)
	if out == "" {
		return "/"
	}
	if out[0] != '/' {
		return path.Join("/", out)
	}
	return out
}

func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Rewrite{}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}
	requestHeadersRewrite := options.RequestHeadersRewrite
	responseHeadersRewrite := options.ResponseHeadersRewrite
	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if options.PathRewrite != nil {
				req.URL.Path = *options.PathRewrite
			}
			if options.HostRewrite != nil {
				req.Host = *options.HostRewrite
			}
			if options.StripPrefix != nil {
				req.URL.Path = stripPrefix(req.URL.Path, options.GetStripPrefix())
			}
			if requestHeadersRewrite != nil {
				for key, value := range requestHeadersRewrite.Set {
					req.Header.Set(key, value)
				}
				for key, value := range requestHeadersRewrite.Add {
					req.Header.Add(key, value)
				}
				for _, value := range requestHeadersRewrite.Remove {
					req.Header.Del(value)

				}
			}
			resp, err := next.RoundTrip(req)
			if err != nil {
				return nil, err
			}
			if responseHeadersRewrite != nil {
				for key, value := range responseHeadersRewrite.Set {
					resp.Header.Set(key, value)
				}
				for key, value := range responseHeadersRewrite.Add {
					resp.Header.Add(key, value)
				}
				for _, value := range responseHeadersRewrite.Remove {
					resp.Header.Del(value)

				}
			}
			return resp, nil
		})
	}, nil
}
