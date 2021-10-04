package proxy

import (
	"fmt"
	"net/http"
	"sync/atomic"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/middleware/cors"
	"github.com/go-kratos/gateway/router"
	"github.com/go-kratos/gateway/router/mux"
)

// Proxy is a gateway proxy.
type Proxy struct {
	router atomic.Value
}

// New new a gateway proxy.
func New() (*Proxy, error) {
	p := &Proxy{}
	p.router.Store(mux.NewRouter())
	return p, nil
}

func (p *Proxy) buildClient(service *config.Service) (client.Client, error) {
	caller, err := client.NewClient(service)
	if err != nil {
		return nil, err
	}
	return caller, nil
}

func (p *Proxy) buildMiddleware(ms []*config.Middleware, handler http.Handler) (http.Handler, error) {
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

// Update updates service endpoint.
func (p *Proxy) Update(services []*config.Service) error {
	router := mux.NewRouter()
	for _, s := range services {
		caller, err := p.buildClient(s)
		if err != nil {
			return err
		}
		for _, e := range s.Endpoints {
			handler, err := p.buildMiddleware(e.Middlewares, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
