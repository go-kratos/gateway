package cors

import (
	"net/http"
	"net/url"
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
	defaultCorsMethods          = []string{"GET", "POST", "PUT", "DELETE"}
	defaultCorsHeaders          = []string{"Origin", "Content-Length", "Content-Type"}
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
	corsMatchAll                  string = "*"
)

func init() {
	middleware.Register("cors", Middleware)
}

func isOriginAllowed(origin string, allowOriginHosts []string) bool {
	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	hostname := strings.ToLower(originURL.Hostname())
	for _, host := range allowOriginHosts {
		host = strings.ToLower(host)
		if host[0] != '*' {
			if hostname == host {
				return true
			}
			continue
		}
		if strings.HasSuffix(hostname, strings.TrimPrefix(host, "*")) {
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
		AllowMethods:        defaultCorsMethods,
		AllowHeaders:        defaultCorsHeaders,
		AllowPrivateNetwork: defaultAllowPrivateNetwork,
		MaxAge:              durationpb.New(time.Minute * 10),
	}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}
	preflightHeaders := generatePreflightHeaders(options)
	normalHeaders := generateNormalHeaders(options)
	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			origin := req.Header.Get(corsOriginHeader)
			if req.Method == corsOptionMethod {
				headers := make(http.Header, len(preflightHeaders)+2)
				if !isOriginAllowed(origin, options.AllowOrigins) {
					return newResponse(http.StatusForbidden, headers)
				}
				if options.AllowPrivateNetwork && req.Header.Get(corsRequestPrivateNetwork) == "true" {
					headers.Set(corsAllowPrivateNetworkHeader, "true")
				}
				for key, value := range preflightHeaders {
					headers[key] = value
				}
				headers.Set(corsAllowOriginHeader, origin)
				return newResponse(http.StatusOK, headers)
			}
			resp, err := next.RoundTrip(req)
			if err != nil {
				return nil, err
			}
			if resp.Header == nil {
				resp.Header = make(http.Header, len(normalHeaders)+1)
			}
			for key, value := range normalHeaders {
				resp.Header[key] = value
			}
			resp.Header.Set(corsAllowOriginHeader, origin)
			return resp, nil
		})
	}, nil
}

func generateNormalHeaders(c *v1.Cors) http.Header {
	headers := make(http.Header)
	if c.AllowCredentials {
		headers.Set(corsAllowCredentialsHeader, "true")
	}
	// backport support for early browsers
	if len(c.AllowMethods) > 0 {
		allowMethods := convert(normalize(c.AllowMethods), strings.ToUpper)
		headers.Set(corsAllowMethodsHeader, strings.Join(allowMethods, ","))
	}
	if len(c.ExposeHeaders) > 0 {
		exposeHeaders := convert(normalize(c.ExposeHeaders), http.CanonicalHeaderKey)
		headers.Set(corsExposeHeadersHeader, strings.Join(exposeHeaders, ","))
	}
	headers.Add(corsVaryHeader, corsOriginHeader)
	return headers
}

func generatePreflightHeaders(c *v1.Cors) http.Header {
	headers := make(http.Header)
	if c.AllowCredentials {
		headers.Set(corsAllowCredentialsHeader, "true")
	}
	if len(c.AllowMethods) > 0 {
		allowMethods := convert(normalize(c.AllowMethods), strings.ToUpper)
		headers.Set(corsAllowMethodsHeader, strings.Join(allowMethods, ","))
	}
	if len(c.AllowHeaders) > 0 {
		allowHeaders := convert(normalize(c.AllowHeaders), http.CanonicalHeaderKey)
		headers.Set(corsAllowHeadersHeader, strings.Join(allowHeaders, ","))
	}
	if c.MaxAge != nil {
		maxAge := int64(c.MaxAge.AsDuration() / time.Second)
		headers.Set(corsMaxAgeHeader, strconv.FormatInt(maxAge, 10))
	}
	// Always set Vary headers
	// see https://github.com/rs/cors/issues/10,
	// https://github.com/rs/cors/commit/dbdca4d95feaa7511a46e6f1efb3b3aa505bc43f#commitcomment-12352001
	headers.Add(corsVaryHeader, corsOriginHeader)
	headers.Add(corsVaryHeader, corsRequestMethodHeader)
	headers.Add(corsVaryHeader, corsRequestHeadersHeader)
	return headers
}

func normalize(values []string) []string {
	if values == nil {
		return nil
	}
	distinctMap := make(map[string]bool, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		value = strings.ToLower(value)
		if _, seen := distinctMap[value]; !seen {
			normalized = append(normalized, value)
			distinctMap[value] = true
		}
	}
	return normalized
}

func convert(s []string, c func(string) string) []string {
	var out []string
	for _, i := range s {
		out = append(out, c(i))
	}
	return out
}
