package cors

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/cors/v1"
	"github.com/go-kratos/gateway/middleware"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	defaultCorsOptionStatusCode = 200
	defaultCorsMethods          = []string{"GET", "HEAD", "POST"}
	defaultCorsHeaders          = []string{"Accept", "Accept-Language", "Content-Language", "Origin"}
	// (WebKit/Safari v9 sends the Origin header by default in AJAX requests)
)

const (
	corsOptionMethod           string = "OPTIONS"
	corsAllowOriginHeader      string = "Access-Control-Allow-Origin"
	corsExposeHeadersHeader    string = "Access-Control-Expose-Headers"
	corsMaxAgeHeader           string = "Access-Control-Max-Age"
	corsAllowMethodsHeader     string = "Access-Control-Allow-Methods"
	corsAllowHeadersHeader     string = "Access-Control-Allow-Headers"
	corsAllowCredentialsHeader string = "Access-Control-Allow-Credentials"
	corsRequestMethodHeader    string = "Access-Control-Request-Method"
	corsRequestHeadersHeader   string = "Access-Control-Request-Headers"
	corsOriginHeader           string = "Origin"
	corsVaryHeader             string = "Vary"
	corsOriginMatchAll         string = "*"
)

func init() {
	middleware.Register("cors", Middleware)
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}
	if len(allowedOrigins) == 0 {
		return true
	}
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == origin || allowedOrigin == corsOriginMatchAll {
			return true
		}
	}
	return false
}

func isMatch(needle string, haystack []string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func newResponse(statusCode int, header http.Header) (*http.Response, error) {
	return &http.Response{StatusCode: statusCode, Header: header}, nil
}

// Middleware automatically sets the allow response header.
func Middleware(cfg *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Cors{
		AllowCredentials: true,
		AllowedMethods:   defaultCorsMethods,
		AllowedHeaders:   defaultCorsHeaders,
		MaxAge:           durationpb.New(time.Minute * 10),
	}
	if err := anypb.UnmarshalTo(cfg.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
		return nil, err
	}
	maxAge := int(options.MaxAge.AsDuration() / time.Second)
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (reply *http.Response, err error) {
			// allowed origins
			header := make(http.Header)
			origin := req.Header.Get(corsOriginHeader)
			if isOriginAllowed(origin, options.AllowedOrigins) {
				if req.Method != corsOptionMethod {
					return handler(ctx, req)
				}
				return newResponse(defaultCorsOptionStatusCode, header)
			}
			if req.Method == corsOptionMethod {
				method := req.Header.Get(corsRequestMethodHeader)
				if method == "" {
					return newResponse(http.StatusBadRequest, header)
				}
				requestHeaders := strings.Split(req.Header.Get(corsRequestHeadersHeader), ",")
				allowedHeaders := []string{}
				for _, v := range requestHeaders {
					canonicalHeader := http.CanonicalHeaderKey(strings.TrimSpace(v))
					if canonicalHeader == "" || isMatch(canonicalHeader, defaultCorsHeaders) {
						continue
					}
					if isMatch(canonicalHeader, options.AllowedHeaders) {
						return newResponse(http.StatusForbidden, header)
					}
					allowedHeaders = append(allowedHeaders, canonicalHeader)
				}
				if len(allowedHeaders) > 0 {
					header.Set(corsAllowHeadersHeader, strings.Join(allowedHeaders, ","))
				}
				if maxAge > 0 {
					header.Set(corsMaxAgeHeader, strconv.Itoa(maxAge))
				}
				if !isMatch(method, defaultCorsMethods) {
					header.Set(corsAllowMethodsHeader, method)
					return newResponse(defaultCorsOptionStatusCode, header)
				}
			} else {
				if len(options.ExposedHeaders) > 0 {
					header.Set(corsExposeHeadersHeader, strings.Join(options.ExposedHeaders, ","))
				}
			}
			if options.AllowCredentials {
				header.Set(corsAllowCredentialsHeader, "true")
			}
			// allowed origins
			if len(options.AllowedOrigins) > 1 {
				header.Set(corsVaryHeader, corsOriginHeader)
			}
			returnOrigin := origin
			if len(options.AllowedOrigins) == 0 {
				returnOrigin = "*"
			} else {
				for _, o := range options.AllowedOrigins {
					// A configuration of * is different than explicitly setting an allowed
					// origin. Returning arbitrary origin headers in an access control allow
					// origin header is unsafe and is not required by any use case.
					if o == corsOriginMatchAll {
						returnOrigin = "*"
						break
					}
				}
			}
			header.Set(corsAllowOriginHeader, returnOrigin)
			// forward cors request
			if req.Method == corsOptionMethod {
				return newResponse(defaultCorsOptionStatusCode, header)
			}
			// invoke next handler
			if reply, err = handler(ctx, req); err == nil {
				for k, v := range header {
					reply.Header[k] = v
				}
			}
			return
		}

	}, nil
}
