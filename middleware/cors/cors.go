package cors

import (
	"net/http"

	"github.com/go-kratos/gateway/middleware"
)

// CORS automatically sets the allow response header.
func CORS() middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// TODO
			next.ServeHTTP(w, req)
		})
	}
}
