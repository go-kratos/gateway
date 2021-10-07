package proxy

import (
	"context"
	"io"
	"log"
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

func (p *Proxy) buildEndpoint(endpoint *config.Endpoint) (http.Handler, error) {
	caller, err := p.clientFactory(endpoint)
	if err != nil {
		return nil, err
	}
	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		opts, _ := FromContext(req.Context())
		ctx, cancel := context.WithTimeout(req.Context(), endpoint.Timeout.AsDuration())
		defer cancel()
		resp, err := caller.Invoke(ctx, req, client.WithLabels(opts.Labels))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		sets := w.Header()
		for k, v := range resp.Header {
			sets[k] = v
		}
		if _, err = io.Copy(w, resp.Body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(resp.StatusCode)
		}
		// see https://pkg.go.dev/net/http#example-ResponseWriter-Trailers
		for k, v := range resp.Trailer {
			sets[k] = v
		}
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
			handler, err := p.buildEndpoint(e)
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
	defer func() {
		if err := recover(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
		}
	}()
	p.router.Load().(router.Router).ServeHTTP(w, r)
}
