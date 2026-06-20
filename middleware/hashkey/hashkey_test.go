package hashkey

import (
	"net/http"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client/consistenthash"
	"github.com/go-kratos/gateway/middleware"
)

func TestHashKeyMiddleware(t *testing.T) {
	c := &config.Middleware{Name: "hashkey"}
	m, err := Middleware(c)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	var extractedKey string
	next := middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		extractedKey = consistenthash.HashKeyFromContext(req.Context())
		return &http.Response{StatusCode: http.StatusOK}, nil
	})

	tripper := m(next)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("X-Session-ID", "session-12345")

	tripper.RoundTrip(req)

	if extractedKey != "session-12345" {
		t.Fatalf("expected hash key to be session-12345, got %s", extractedKey)
	}
}
