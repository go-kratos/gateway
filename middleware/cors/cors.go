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
	defaultAllowCredentials     = true
	defaultAllowPrivateNetwork  = false
	defaultCorsOptionStatusCode = 200
	defaultCorsMethods          = []string{"GET", "HEAD", "POST"}
	defaultCorsHeaders          = []string{"Accept", "Accept-Language", "Content-Language", "Origin"}
	// (WebKit/Safari v9 sends the Origin header by default in AJAX requests)
)

const (
	corsOptionMethod              string = "OPTIONS"
	corsAllowOriginHeader         string = "Access-Control-Allow-Origin"
	corsExposeHeadersHeader       string = "Access-Control-Expose-Headers"
	corsMaxAgeHeader              string = "Access-Control-Max-Age"
	corsAllowMethodsHeader        string = "Access-Control-Allow-Methods"
	corsAllowHeadersHeader        string = "Access-Control-Allow-Headers"
	corsAllowCredentialsHeader    string = "Access-Control-Allow-Credentials"
	corsAllowPrivateNetworkHeader string = "Access-Control-Allow-Private-Network"
	corsRequestMethodHeader       string = "Access-Control-Request-Method"
	corsRequestHeadersHeader      string = "Access-Control-Request-Headers"
	corsRequestPrivateNetwork     string = "Access-Control-Request-Private-Network"
	corsOriginHeader              string = "Origin"
	corsVaryHeader                string = "Vary"
	corsOriginMatchAll            string = "*"
	corsHeaderMatchAll            string = "*"
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

func isMethodAllowed(method string, allowedMethods []string) bool {
	if method == "" || len(allowedMethods) == 0 {
		return false
	}
	// Always allow preflight requests
	if method == corsOptionMethod {
		return true
	}

	for _, allowedMethod := range allowedMethods {
		if allowedMethod == method {
			return true
		}
	}
	return false
}

func areHeadersAllowed(requestedHeaders []string, allowedHeaders []string) bool {
	if len(requestedHeaders) == 0 {
		return true
	}
LOOP:
	for i, requestedHeader := range requestedHeaders {
		canonicalHeader := http.CanonicalHeaderKey(strings.TrimSpace(requestedHeader))

		for _, allowedHeader := range allowedHeaders {
			if canonicalHeader == allowedHeader || allowedHeader == corsHeaderMatchAll {
				requestedHeaders[i] = canonicalHeader
				continue LOOP
			}
		}
		return false
	}
	return true
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
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Cors{
		AllowCredentials:    defaultAllowCredentials,
		AllowedMethods:      defaultCorsMethods,
		AllowedHeaders:      defaultCorsHeaders,
		AllowPrivateNetwork: defaultAllowPrivateNetwork,
		MaxAge:              durationpb.New(time.Minute * 10),
	}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}
	maxAge := int(options.MaxAge.AsDuration() / time.Second)
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (reply *http.Response, err error) {
			header := make(http.Header)
			origin := req.Header.Get(corsOriginHeader)
			// should handle preflight requests first
			if req.Method == corsOptionMethod {
				method := req.Header.Get(corsRequestMethodHeader)

				// Always set Vary headers
				header.Add(corsVaryHeader, corsOriginHeader)
				header.Add(corsVaryHeader, corsRequestMethodHeader)
				header.Add(corsVaryHeader, corsRequestHeadersHeader)
				if !isOriginAllowed(origin, options.AllowedOrigins) {
					return newResponse(http.StatusBadRequest, header)
				}
				if !isMethodAllowed(method, options.AllowedMethods) {
					return newResponse(http.StatusBadRequest, header)
				}
				requestHeaders := strings.Split(req.Header.Get(corsRequestHeadersHeader), ",")
				if !areHeadersAllowed(requestHeaders, options.AllowedHeaders) {
					return newResponse(http.StatusBadRequest, header)
				}
				returnOrigin := origin
				for _, o := range options.AllowedOrigins {
					// A configuration of * is different than explicitly setting an allowed
					// origin. Returning arbitrary origin headers in an access control allow
					// origin header is unsafe and is not required by any use case.
					if o == corsOriginMatchAll {
						returnOrigin = "*"
						break
					}
				}
				header.Set(corsAllowOriginHeader, returnOrigin)
				header.Set(corsAllowMethodsHeader, method)
				if len(requestHeaders) > 0 {
					// Spec says: Since the list of headers can be unbounded, simply returning supported headers
					// from Access-Control-Request-Headers can be enough
					header.Set(corsAllowHeadersHeader, strings.Join(requestHeaders, ", "))
				}
				if options.AllowCredentials {
					header.Set(corsAllowCredentialsHeader, "true")
				}
				if maxAge > 0 {
					header.Set(corsMaxAgeHeader, strconv.Itoa(maxAge))
				}
				if req.Header.Get(corsRequestPrivateNetwork) == "true" && options.AllowPrivateNetwork {
					header.Set(corsAllowPrivateNetworkHeader, "true")
				}
				return newResponse(defaultCorsOptionStatusCode, header)
			} else {
				// Always set Vary headers
				header.Add(corsVaryHeader, corsOriginHeader)
				if isOriginAllowed(origin, options.AllowedOrigins) {
					returnOrigin := origin
					for _, o := range options.AllowedOrigins {
						// A configuration of * is different than explicitly setting an allowed
						// origin. Returning arbitrary origin headers in an access control allow
						// origin header is unsafe and is not required by any use case.
						if o == corsOriginMatchAll {
							returnOrigin = "*"
							break
						}
					}
					header.Set(corsAllowOriginHeader, returnOrigin)
				}

				if len(options.ExposedHeaders) > 0 {
					header.Set(corsExposeHeadersHeader, strings.Join(options.ExposedHeaders, ","))
				}
				if options.AllowCredentials {
					header.Set(corsAllowCredentialsHeader, "true")
				}

				// invoke next handler
				if reply, err = handler(ctx, req); err == nil {
					for k, v := range header {
						reply.Header[k] = v
					}
				}
				return
			}
		}
	}, nil
}
