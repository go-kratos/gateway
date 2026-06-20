package auth

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

func init() {
	middleware.Register("auth", Middleware)
}

func newResponse(statusCode int, header http.Header) (*http.Response, error) {
	return &http.Response{
		StatusCode: statusCode,
		Header:     header,
		Body:       io.NopCloser(&bytes.Buffer{}),
	}, nil
}

// Middleware implements simple API Key authentication.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	// Without protoc, we hardcode the valid keys for demonstration
	validKeys := map[string]bool{
		"secret-key-123": true,
		"admin-key-456":  true,
	}

	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			authHeader := req.Header.Get("Authorization")
			if authHeader == "" {
				return newResponse(http.StatusUnauthorized, http.Header{})
			}

			// Expecting "Bearer <key>" or just "<key>"
			parts := strings.SplitN(authHeader, " ", 2)
			var key string
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				key = parts[1]
			} else {
				key = authHeader
			}

			if !validKeys[key] {
				return newResponse(http.StatusUnauthorized, http.Header{})
			}

			return next.RoundTrip(req)
		})
	}, nil
}
