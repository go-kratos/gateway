package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"runtime"
	"sync/atomic"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/router"
	"github.com/go-kratos/gateway/router/mux"
	"github.com/go-kratos/kratos/v2/log"
	gorillamux "github.com/gorilla/mux"
)

const xff = "X-Forwarded-For"

// LOG .
var LOG = log.NewHelper(log.With(log.GetLogger(), "source", "proxy"))

// Proxy is a gateway proxy.
type Proxy struct {
	router            atomic.Value
	clientFactory     client.Factory
	middlewareFactory middleware.Factory
}

// New is new a gateway proxy.
func New(clientFactory client.Factory, middlewareFactory middleware.Factory) (*Proxy, error) {
	p := &Proxy{
		clientFactory:     clientFactory,
		middlewareFactory: middlewareFactory,
	}
	p.router.Store(mux.NewRouter())
	return p, nil
}

func (p *Proxy) buildMiddleware(ms []*config.Middleware, handler middleware.Handler) (middleware.Handler, error) {
	for i := len(ms) - 1; i >= 0; i-- {
		m, err := p.middlewareFactory(ms[i])
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
	handler, err := p.buildMiddleware(ms, caller.Do)
	if err != nil {
		return nil, err
	}
	handler, err = p.buildMiddleware(e.Middlewares, handler)
	if err != nil {
		return nil, err
	}
	return http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			r.Header[xff] = append(r.Header[xff], ip)
		}
		ctx := middleware.NewRequestContext(r.Context(), &middleware.RequestOptions{})
		ctx, cancel := context.WithTimeout(ctx, e.Timeout.AsDuration())
		defer cancel()
		resp, err := handler(ctx, r)
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
		headers := w.Header()
		for k, v := range resp.Header {
			headers[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		if body := resp.Body; body != nil {
			_, _ = io.Copy(w, body)
		}
		// see https://pkg.go.dev/net/http#example-ResponseWriter-Trailers
		for k, v := range resp.Trailer {
			headers[http.TrailerPrefix+k] = v
		}
		resp.Body.Close()
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
		LOG.Infof("build endpoint: [%s] %s %s", e.Protocol, e.Method, e.Path)
	}
	p.router.Store(router)
	return nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			w.WriteHeader(http.StatusBadGateway)
			buf := make([]byte, 64<<10) //nolint:gomnd
			n := runtime.Stack(buf, false)
			LOG.Errorf("panic recovered: %s", buf[:n])
		}
	}()
	p.router.Load().(router.Router).ServeHTTP(w, req)
}

func (p *Proxy) DebugHandler() http.Handler {
	debugMux := gorillamux.NewRouter()
	debugMux.Methods("GET").Path("/_/debug/proxy/router/inspect").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		router, ok := p.router.Load().(router.Router)
		if !ok {
			return
		}
		inspect := mux.InspectMuxRouter(router)
		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(inspect)
	})
	return debugMux
}
