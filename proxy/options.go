package proxy

import (
	"fmt"
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/middleware/cors"
)

// ClientFactory is returns service client.
type ClientFactory func(service *config.Service) (client.Client, error)

// MiddlewareFactory is returns middleware handler.
type MiddlewareFactory func(ms []*config.Middleware, handler http.Handler) (http.Handler, error)

// Option is proxy option func.
type Option func(*Proxy)

// WithMiddewareFactory with middleware factory.
func WithMiddewareFactory(f MiddlewareFactory) Option {
	return func(o *Proxy) {
		o.middlewareFactory = f
	}
}

func defaultMiddlewareFactory(ms []*config.Middleware, handler http.Handler) (http.Handler, error) {
	for _, m := range ms {
		switch m.Name {
		case cors.Name:
			handler = cors.Middleware(m)(handler)
		default:
			return nil, fmt.Errorf("not found middleware: %s", m.Name)
		}
	}
	return handler, nil
}
