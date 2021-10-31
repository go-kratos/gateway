package cors

import (
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/cors/v1"
	"github.com/go-kratos/gateway/endpoint"
)

// Name is the middleware name.
const Name = "cors"

// Middleware automatically sets the allow response header.
func Middleware(cfg *config.Middleware) (endpoint.Middleware, error) {
	options := &v1.Cors{}
	if err := cfg.Options.UnmarshalTo(options); err != nil {
		return nil, err
	}

	opts := make([]CORSOption, 0, 6)
	if len(options.AllowedHeaders) > 0 {
		opts = append(opts, AllowedHeaders(options.AllowedHeaders))
	}
	if len(options.AllowedMethods) > 0 {
		opts = append(opts, AllowedMethods(options.AllowedMethods))
	}
	if len(options.AllowedOrigins) > 0 {
		opts = append(opts, AllowedOrigins(options.AllowedOrigins))
	}
	if len(options.ExposedHeaders) > 0 {
		opts = append(opts, ExposedHeaders(options.ExposedHeaders))
	}
	if options.MaxAge != nil {
		maxAge := int(options.MaxAge.AsDuration() / time.Second)
		if maxAge > 0 {
			opts = append(opts, MaxAge(maxAge))
		}
	}
	if options.AllowCredentials {
		opts = append(opts, AllowCredentials())
	}

	corsMiddleware := CORS(opts...)
	return func(handler endpoint.Endpoint) endpoint.Endpoint {
		return corsMiddleware(handler)
	}, nil
}
