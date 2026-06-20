package cache

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

func init() {
	middleware.Register("cache", Middleware)
}

type cacheEntry struct {
	statusCode int
	headers    http.Header
	body       []byte
	expiresAt  time.Time
}

var (
	memoryCache sync.Map
	defaultTTL  = 60 * time.Second
)

// Middleware implements an in-memory GET request cache.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	// Without protoc, we hardcode the config
	enabled := true
	ttl := defaultTTL

	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if !enabled || req.Method != http.MethodGet {
				return next.RoundTrip(req)
			}

			cacheKey := req.URL.Path + "?" + req.URL.RawQuery

			// Check Cache
			if val, ok := memoryCache.Load(cacheKey); ok {
				entry := val.(cacheEntry)
				if time.Now().Before(entry.expiresAt) {
					// Cache Hit
					headers := make(http.Header)
					for k, v := range entry.headers {
						headers[k] = v
					}
					headers.Set("X-Cache", "HIT")
					return &http.Response{
						StatusCode: entry.statusCode,
						Header:     headers,
						Body:       io.NopCloser(bytes.NewBuffer(entry.body)),
					}, nil
				}
				// Expired, delete
				memoryCache.Delete(cacheKey)
			}

			// Cache Miss
			resp, err := next.RoundTrip(req)
			if err != nil {
				return nil, err
			}

			// Only cache 200 OK
			if resp.StatusCode == http.StatusOK && resp.Body != nil {
				bodyBytes, err := io.ReadAll(resp.Body)
				if err == nil {
					// Store in cache
					headersCopy := make(http.Header)
					for k, v := range resp.Header {
						headersCopy[k] = v
					}
					memoryCache.Store(cacheKey, cacheEntry{
						statusCode: resp.StatusCode,
						headers:    headersCopy,
						body:       bodyBytes,
						expiresAt:  time.Now().Add(ttl),
					})
					// Restore body for the client
					resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}

			if resp.Header == nil {
				resp.Header = make(http.Header)
			}
			resp.Header.Set("X-Cache", "MISS")
			return resp, nil
		})
	}, nil
}
