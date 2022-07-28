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
			} else if options.GetStripPrefix() > 0 {
				req.URL.Path = stripPrefixReqPath(options.GetStripPrefix(), req.URL.Path)
			}
			if requestHeadersRewrite != nil {
				for key, value := range requestHeadersRewrite.Set {
					req.Header.Set(key, value)
				}
				for key, value := range requestHeadersRewrite.Add {
					if req.Header.Get(key) == "" {
						req.Header.Add(key, value)
					} else {
						req.Header.Set(key, value)
					}
				}
				for _, value := range requestHeadersRewrite.Remove {
					if req.Header.Get(value) != "" {
						req.Header.Del(value)
					}
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
					if resp.Header.Get(key) == "" {
						resp.Header.Add(key, value)
					} else {
						resp.Header.Set(key, value)
					}
				}
				for _, value := range responseHeadersRewrite.Remove {
					if resp.Header.Get(value) != "" {
						resp.Header.Del(value)
					}
				}
			}
			return resp, nil
		})
	}, nil
}

func stripPrefixReqPath(stripPrefix int64, path string) string {
	if stripPrefix > 0 {
		var newPath string
		parts := splitIgnoreEmpty(path, "/")
		for i := range parts {
			if int64(i) < stripPrefix {
				continue
			}
			part := parts[i]
			newPath += "/" + part
		}
		if strings.HasSuffix(path, "/") {
			newPath = newPath + "/"
		}
		return newPath
	}
	return path
}

func splitIgnoreEmpty(path, delimiter string) []string {
	parts := strings.Split(path, delimiter)
	newParts := make([]string, 0)
	for i := range parts {
		part := parts[i]
		if len(part) > 0 {
			newParts = append(newParts, part)
		}
	}
	return newParts
}
