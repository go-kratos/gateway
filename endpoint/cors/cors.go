package cors

import (
	"net/http"

	"github.com/go-kratos/gateway/endpoint"
)

// CORS automatically sets the allow response header.
func CORS() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// TODO
			next.ServeHTTP(w, req)
		})
	}
}
