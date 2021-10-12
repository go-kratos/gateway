package proxy

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/endpoint"
	"github.com/go-kratos/gateway/router"
	"github.com/go-kratos/gateway/router/mux"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/selector"
)

// ClientFactory is returns service client.
type ClientFactory func(endpoint *config.Endpoint) (client.Client, error)

// MiddlewareFactory is returns middleware handler.
type MiddlewareFactory func(*config.Middleware) (endpoint.Middleware, error)

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

func (p *Proxy) buildMiddleware(ms []*config.Middleware, handler endpoint.Endpoint) (endpoint.Endpoint, error) {
	for _, c := range ms {
		m, err := p.middlewareFactory(c)
		if err != nil {
			return nil, err
		}
		handler = m(handler)
	}
	return handler, nil
}

func (p *Proxy) buildEndpoint(e *config.Endpoint, ms []*config.Middleware) (http.Handler, error) {
	caller, err := p.clientFactory(e)
	if err != nil {
		return nil, err
	}
	handler, err := p.buildMiddleware(ms, caller.Invoke)
	if err != nil {
		return nil, err
	}
	handler, err = p.buildMiddleware(e.Middlewares, handler)
	if err != nil {
		return nil, err
	}
	return http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), e.Timeout.AsDuration())
		defer cancel()
		req := endpoint.NewRequest(r)
		defer endpoint.FreeRequest(req)
		resp, err := handler(ctx, req)
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
		defer resp.Body().Close()
		defer endpoint.FreeResponse(resp)
		headers := w.Header()
		for k, v := range resp.Header() {
			headers[k] = v
		}
		w.WriteHeader(resp.StatusCode())
		if body := resp.Body(); body != nil {
			_, _ = io.Copy(w, body)
		}
		// see https://pkg.go.dev/net/http#example-ResponseWriter-Trailers
		for k, v := range resp.Trailer() {
			headers[http.TrailerPrefix+k] = v
		}
	})), nil
}

// Update updates service endpoint.
func (p *Proxy) Update(c *config.Gateway) error {
	router := mux.NewRouter()
	for _, e := range c.Endpoints {
		handler, err := p.buildEndpoint(e, c.Middlewares)
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
	ctx := endpoint.NewContext(req.Context(), &endpoint.RequestOptions{
		Filters: []selector.Filter{},
	})
	p.router.Load().(router.Router).ServeHTTP(w, req.WithContext(ctx))
}
