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
	"github.com/go-kratos/kratos/v2/selector"
)

// ClientFactory is returns service client.
type ClientFactory func(endpoint *config.Endpoint) (client.Client, error)

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
		ctx, cancel := context.WithTimeout(req.Context(), endpoint.Timeout.AsDuration())
		defer cancel()
		opts, _ := FromContext(req.Context())
		resp, err := caller.Invoke(ctx, req, client.WithFilter(opts.Filters))
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		sets := w.Header()
		for k, v := range resp.Header {
			sets[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
		// see https://pkg.go.dev/net/http#example-ResponseWriter-Trailers
		for k, v := range resp.Trailer {
			sets[http.TrailerPrefix+k] = v
		}
	}))
	return p.buildMiddleware(endpoint.Middlewares, handler)
}

func (p *Proxy) buildMiddleware(ms []*config.Middleware, handler http.Handler) (http.Handler, error) {
	for _, c := range ms {
		m, err := p.middlewareFactory(c)
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
			handler, err = p.buildMiddleware(s.Middlewares, handler)
			if err != nil {
				return err
			}
			if err = router.Handle(e.Path, e.Method, handler); err != nil {
				return err
			}
			log.Printf("build endpoint: [%s] %s %s\n", e.Protocol, e.Method, e.Path)
		}
	}
	p.router.Store(router)
	return nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
		}
	}()
	ctx := NewContext(req.Context(), &RequestOptions{
		Filters: []selector.Filter{},
	})
	p.router.Load().(router.Router).ServeHTTP(w, req.WithContext(ctx))
}
