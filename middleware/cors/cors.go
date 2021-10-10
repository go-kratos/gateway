package cors

import (
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/cors/v1"
	"github.com/go-kratos/gateway/middleware"
)

// Name is the middleware name.
const Name = "cors"

// Middleware automatically sets the allow response header.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Cors{}
	if err := c.Options.UnmarshalTo(options); err != nil {
		return nil, err
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// TODO
			next.ServeHTTP(w, req)
		})
	}, nil
}
