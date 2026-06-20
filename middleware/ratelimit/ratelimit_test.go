package ratelimit

import (
	"net/http"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

func TestRateLimitMiddleware(t *testing.T) {
	c := &config.Middleware{Name: "ratelimit"}
	m, err := Middleware(c)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	next := middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK}, nil
	})

	tripper := m(next)

	// Hardcoded in middleware: rps=5, burst=10.
	// So 10 immediate requests should pass. The 11th should fail.
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", "http://example.com/api", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		resp, _ := tripper.RoundTrip(req)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d should have passed, got %d", i+1, resp.StatusCode)
		}
	}

	// 11th request from same IP should fail
	reqFail, _ := http.NewRequest("GET", "http://example.com/api", nil)
	reqFail.RemoteAddr = "192.168.1.1:54321" // same IP, different port
	respFail, _ := tripper.RoundTrip(reqFail)
	if respFail.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("request 11 should have failed with 429, got %d", respFail.StatusCode)
	}

	// Request from a different IP should pass
	reqDiff, _ := http.NewRequest("GET", "http://example.com/api", nil)
	reqDiff.RemoteAddr = "10.0.0.1:12345"
	respDiff, _ := tripper.RoundTrip(reqDiff)
	if respDiff.StatusCode != http.StatusOK {
		t.Fatalf("request from different IP should have passed, got %d", respDiff.StatusCode)
	}
}
