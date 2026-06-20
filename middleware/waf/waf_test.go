package waf

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

func TestWafMiddleware(t *testing.T) {
	c := &config.Middleware{
		Name:    "waf",
	}

	m, err := Middleware(c)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	next := middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK}, nil
	})

	tripper := m(next)

	tests := []struct {
		name       string
		method     string
		url        string
		body       string
		wantStatus int
	}{
		{
			name:       "valid request",
			method:     "GET",
			url:        "http://example.com/api?user=123",
			body:       "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "payload too large",
			method:     "POST",
			url:        "http://example.com/api",
			body:       "this is a very long body that exceeds ten bytes",
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "sqli in query",
			method:     "GET",
			url:        "http://example.com/api?id=1 UNION SELECT * FROM users",
			body:       "",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "xss in body",
			method:     "POST",
			url:        "http://example.com/api",
			body:       "<script>alert(1)</script>",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != "" {
				body = bytes.NewBufferString(tt.body)
			}
			req, err := http.NewRequest(tt.method, tt.url, body)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			if tt.body != "" {
				req.ContentLength = int64(len(tt.body))
			}

			resp, err := tripper.RoundTrip(req)
			if err != nil {
				t.Fatalf("roundtrip failed: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("got status %v, want %v", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}
