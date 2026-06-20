package hashkey

import (
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client/consistenthash"
	"github.com/go-kratos/gateway/middleware"
)

func init() {
	middleware.Register("hashkey", Middleware)
}

// Middleware injects the X-Session-ID as the consistent hashing key.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			sessionID := req.Header.Get("X-Session-ID")
			// Inject into context for the selector to use
			ctx := consistenthash.WithHashKey(req.Context(), sessionID)
			req = req.WithContext(ctx)
			return next.RoundTrip(req)
		})
	}, nil
}
