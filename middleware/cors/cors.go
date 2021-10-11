package cors

import (
	"net/http"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/cors/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/gorilla/handlers"
	"github.com/pkg/errors"
)

// Name is the middleware name.
const Name = "cors"

// Middleware automatically sets the allow response header.
func Middleware(cfg *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Cors{}
	if err := cfg.Options.UnmarshalTo(options); err != nil {
		return nil, errors.WithStack(err)
	}

	opts := make([]handlers.CORSOption, 0, 6)
	if len(options.AllowedHeaders) > 0 {
		opts = append(opts, handlers.AllowedHeaders(options.AllowedHeaders))
	}
	if len(options.AllowedMethods) > 0 {
		opts = append(opts, handlers.AllowedMethods(options.AllowedMethods))
	}
	if len(options.AllowedOrigins) > 0 {
		opts = append(opts, handlers.AllowedOrigins(options.AllowedOrigins))
	}
	if len(options.ExposedHeaders) > 0 {
		opts = append(opts, handlers.ExposedHeaders(options.ExposedHeaders))
	}
	if options.MaxAge != nil {
		maxAge := int(options.MaxAge.AsDuration() / time.Second)
		if maxAge > 0 {
			opts = append(opts, handlers.MaxAge(maxAge))
		}
	}
	if options.AllowCredentials {
		opts = append(opts, handlers.AllowCredentials())
	}

	corsMiddleware := handlers.CORS(opts...)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			corsMiddleware(next).ServeHTTP(w, req)
		})
	}, nil
}
