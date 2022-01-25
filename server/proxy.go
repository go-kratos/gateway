package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var (
	// LOG .
	LOG = log.NewHelper(log.With(log.GetLogger(), "source", "server"))

	defaultTimeout     = time.Second * 15
	defaultIdleTimeout = time.Second * 300
)

// ProxyServer is a proxy server.
type ProxyServer struct {
	*http.Server
}

// NewProxy new a gateway server.
func NewProxy(handler http.Handler, addr string, timeout time.Duration, idleTimeout time.Duration) *ProxyServer {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if idleTimeout <= 0 {
		idleTimeout = defaultIdleTimeout
	}
	return &ProxyServer{
		Server: &http.Server{
			Addr: addr,
			Handler: h2c.NewHandler(handler, &http2.Server{
				IdleTimeout: idleTimeout,
			}),
			ReadTimeout:       timeout,
			ReadHeaderTimeout: timeout,
			WriteTimeout:      timeout,
			IdleTimeout:       idleTimeout,
		},
	}
}

// Start start the server.
func (s *ProxyServer) Start(ctx context.Context) error {
	LOG.Infof("proxy server listening on %s", s.Addr)
	return s.ListenAndServe()
}

// Stop stop the server.
func (s *ProxyServer) Stop(ctx context.Context) error {
	LOG.Info("proxy server stopping")
	return s.Shutdown(ctx)
}
