package proxy

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/router"
	"github.com/go-kratos/gateway/router/mux"
)

// ClientFactory is returns service client.
type ClientFactory func(*config.Endpoint) (client.Client, error)

// MiddlewareFactory is returns middleware handler.
type MiddlewareFactory func(*config.Middleware) (middleware.Middleware, error)

// Proxy is a gateway proxy.
type Proxy struct {
	router            atomic.Value
	clientFactory     ClientFactory
	middlewareFactory MiddlewareFactory
}

// New new a gateway proxy.
func New(clientFactory ClientFactory, middlewareFactory MiddlewareFactory) (*Proxy, error) {
	p := &Proxy{
		clientFactory:     clientFactory,
		middlewareFactory: middlewareFactory,
	}
	p.router.Store(mux.NewRouter())
	return p, nil
}

func (p *Proxy) buildEndpoint(caller client.Client, endpoint *config.Endpoint) (http.Handler, error) {
	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), endpoint.Timeout.AsDuration())
		defer cancel()
		resp, err := caller.Invoke(ctx, req)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		defer resp.Body.Close()
		header := w.Header()
		for k, v := range resp.Header {
			header[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		_, err = io.Copy(w, resp.Body)
	}))
	for _, mc := range endpoint.Middlewares {
		m, err := p.middlewareFactory(mc)
		if err != nil {
			return nil, err
		}
		handler = m(handler)
	}
	return handler, nil
}

// Update updates service endpoint.
func (p *Proxy) Update(services []*config.Service) error {
	router := mux.NewRouter()
	for _, s := range services {
		for _, e := range s.Endpoints {
			caller, err := p.clientFactory(e)
			if err != nil {
				return err
			}
			handler, err := p.buildEndpoint(caller, e)
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
