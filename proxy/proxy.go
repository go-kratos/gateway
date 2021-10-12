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
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/selector"
)

// ClientFactory is returns service client.
type ClientFactory func(endpoint *config.Endpoint) (client.Client, error)

// MiddlewareFactory is returns middleware handler.
type MiddlewareFactory func(*config.Middleware) (middleware.Middleware, error)

// Proxy is a gateway proxy.
type Proxy struct {
	router            atomic.Value
	log               *log.Helper
	clientFactory     ClientFactory
	middlewareFactory MiddlewareFactory
}

// New new a gateway proxy.
func New(logger log.Logger, clientFactory ClientFactory, middlewareFactory MiddlewareFactory) (*Proxy, error) {
	p := &Proxy{
		log:               log.NewHelper(logger),
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
			switch err {
			case context.Canceled:
				w.WriteHeader(499)
			case context.DeadlineExceeded:
				w.WriteHeader(504)
			default:
				w.WriteHeader(502)
			}
			return
		}
		defer resp.Body.Close()
		headers := w.Header()
		for k, v := range resp.Header {
			headers[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
		// see https://pkg.go.dev/net/http#example-ResponseWriter-Trailers
		for k, v := range resp.Trailer {
			headers[http.TrailerPrefix+k] = v
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
func (p *Proxy) Update(c *config.Gateway) error {
	router := mux.NewRouter()
	for _, e := range c.Endpoints {
		handler, err := p.buildEndpoint(e)
		if err != nil {
			return err
		}
		handler, err = p.buildMiddleware(c.Middlewares, handler)
		if err != nil {
			return err
		}
		if err = router.Handle(e.Path, e.Method, handler); err != nil {
			return err
		}
		p.log.Infof("build endpoint: [%s] %s %s", e.Protocol, e.Method, e.Path)
	}
	p.router.Store(router)
	return nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			w.WriteHeader(http.StatusBadGateway)
			p.log.Error(err)
		}
	}()
	ctx := NewContext(req.Context(), &RequestOptions{
		Filters: []selector.Filter{},
	})
	p.router.Load().(router.Router).ServeHTTP(w, req.WithContext(ctx))
}
