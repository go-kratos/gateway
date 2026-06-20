package ratelimit

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

func init() {
	middleware.Register("ratelimit", Middleware)
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	limiters = make(map[string]*clientLimiter)
	mu       sync.Mutex
)

func init() {
	// Background cleanup routine for stale limiters
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, cl := range limiters {
				if time.Since(cl.lastSeen) > 3*time.Minute {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()
}

func getLimiter(ip string, rps rate.Limit, burst int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(rps, burst)
		limiters[ip] = &clientLimiter{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func getClientIP(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return ip
}

// Middleware implements traditional Token Bucket rate limiting.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	// Without protoc we use hardcoded values
	// e.g. 5 requests per second, burst of 10
	var rps rate.Limit = 5
	burst := 10

	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			ip := getClientIP(req)
			limiter := getLimiter(ip, rps, burst)

			if !limiter.Allow() {
				return &http.Response{
					Status:     http.StatusText(http.StatusTooManyRequests),
					StatusCode: http.StatusTooManyRequests,
					Header:     http.Header{},
					Body:       io.NopCloser(&bytes.Buffer{}),
				}, nil
			}

			return next.RoundTrip(req)
		})
	}, nil
}
