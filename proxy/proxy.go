package proxy

import (
	"net/http"
	"sync/atomic"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/router"
	"github.com/go-kratos/gateway/router/mux"
)

// ClientFactory is returns service client.
type ClientFactory func(service *config.Service) (client.Client, error)

// MiddlewareFactory is returns middleware handler.
type MiddlewareFactory func(ms []*config.Middleware, handler http.Handler) (http.Handler, error)

// Option is proxy option func.
type Option func(*Proxy)

// WithClientFactory with client factory.
func WithClientFactory(f ClientFactory) Option {
	return func(o *Proxy) {
		o.clientFactory = f
	}
}

// WithMiddewareFactory with middleware factory.
func WithMiddewareFactory(f MiddlewareFactory) Option {
	return func(o *Proxy) {
		o.middlewareFactory = f
	}
}

// Proxy is a gateway proxy.
type Proxy struct {
	router            atomic.Value
	clientFactory     ClientFactory
	middlewareFactory MiddlewareFactory
}

// New new a gateway proxy.
func New() (*Proxy, error) {
	p := &Proxy{}
	p.router.Store(mux.NewRouter())
	return p, nil
}

// Update updates service endpoint.
func (p *Proxy) Update(services []*config.Service) error {
	router := mux.NewRouter()
	for _, s := range services {
		caller, err := p.clientFactory(s)
		if err != nil {
			return err
		}
		for _, e := range s.Endpoints {
			handler, err := p.middlewareFactory(e.Middlewares, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				caller.Invoke(w, req)
			}))
			if err != nil {
				return err
			}
			router.Handle(e.Path, e.Method, handler)
		}
	}
	p.router.Store(router)
	return nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.router.Load().(router.Router).ServeHTTP(w, r)
}
