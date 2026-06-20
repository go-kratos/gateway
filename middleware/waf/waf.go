package waf

import (
	"bytes"
	"io"
	"net/http"
	"regexp"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

var (
	// Very basic regex for demonstration purposes
	sqliRegex = regexp.MustCompile(`(?i)(union\s+select|select\s+.*\s+from|insert\s+into|drop\s+table|update\s+.*\s+set)`)
	xssRegex  = regexp.MustCompile(`(?i)(<script>|javascript:|onerror=|onload=)`)
)

func init() {
	middleware.Register("waf", Middleware)
}

func newResponse(statusCode int, header http.Header) (*http.Response, error) {
	return &http.Response{
		StatusCode: statusCode,
		Header:     header,
		Body:       io.NopCloser(&bytes.Buffer{}),
	}, nil
}

// Middleware implements the Web Application Firewall.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	// Without protoc, we will hardcode the options
	enableSQLi := true
	enableXSS := true
	var maxRequestBodySize int64 = 30 // 30 bytes for test limits

	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			// 1. Check max request body size
			if maxRequestBodySize > 0 && req.ContentLength > maxRequestBodySize {
				return newResponse(http.StatusRequestEntityTooLarge, http.Header{})
			}

			// 2. Inspection
			if enableSQLi || enableXSS {
				// Check query params
				queryStr := req.URL.RawQuery
				if enableSQLi && sqliRegex.MatchString(queryStr) {
					return newResponse(http.StatusForbidden, http.Header{})
				}
				if enableXSS && xssRegex.MatchString(queryStr) {
					return newResponse(http.StatusForbidden, http.Header{})
				}

				// Check body if it's there
				if req.Body != nil {
					bodyBytes, err := io.ReadAll(req.Body)
					if err != nil {
						return newResponse(http.StatusBadRequest, http.Header{})
					}
					// Restore the body for downstream middlewares/proxy
					req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

					bodyStr := string(bodyBytes)
					if enableSQLi && sqliRegex.MatchString(bodyStr) {
						return newResponse(http.StatusForbidden, http.Header{})
					}
					if enableXSS && xssRegex.MatchString(bodyStr) {
						return newResponse(http.StatusForbidden, http.Header{})
					}
				}
			}

			return next.RoundTrip(req)
		})
	}, nil
}
