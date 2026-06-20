package cache

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

func TestCacheMiddleware(t *testing.T) {
	c := &config.Middleware{Name: "cache"}
	m, err := Middleware(c)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	callCount := 0
	next := middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("hello cache")),
		}, nil
	})

	tripper := m(next)

	req1, _ := http.NewRequest("GET", "http://example.com/api/data?id=1", nil)
	resp1, _ := tripper.RoundTrip(req1)
	if resp1.Header.Get("X-Cache") != "MISS" {
		t.Errorf("Expected MISS on first request")
	}

	// Read body so it simulates actual client read
	io.ReadAll(resp1.Body)

	// Second request should be a hit
	req2, _ := http.NewRequest("GET", "http://example.com/api/data?id=1", nil)
	resp2, _ := tripper.RoundTrip(req2)
	if resp2.Header.Get("X-Cache") != "HIT" {
		t.Errorf("Expected HIT on second request")
	}
	body2, _ := io.ReadAll(resp2.Body)
	if string(body2) != "hello cache" {
		t.Errorf("Expected 'hello cache', got '%s'", string(body2))
	}

	if callCount != 1 {
		t.Errorf("Backend should only be called once, called %d times", callCount)
	}

	// Different URL should MISS
	req3, _ := http.NewRequest("GET", "http://example.com/api/data?id=2", nil)
	resp3, _ := tripper.RoundTrip(req3)
	if resp3.Header.Get("X-Cache") != "MISS" {
		t.Errorf("Expected MISS for different query param")
	}

	// POST request should not cache
	reqPost, _ := http.NewRequest("POST", "http://example.com/api/data?id=1", nil)
	tripper.RoundTrip(reqPost)
	if callCount != 3 { // 1 for GET id=1, 1 for GET id=2, 1 for POST
		t.Errorf("Expected 3 total backend calls, got %d", callCount)
	}
}
