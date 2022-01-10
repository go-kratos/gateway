package server

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var (
	LOG = log.NewHelper(log.With(log.GetLogger(), "source", "server"))
)

// Server is a gateway server.
type Server struct {
	*http.Server
}

// New new a gateway server.
func New(handler http.Handler, addr string, timeout time.Duration, idleTimeout time.Duration) *Server {
	srv := &Server{
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
	return srv
}

// Start start the server.
func (s *Server) Start(ctx context.Context) error {
	LOG.Infof("server listening on %s", s.Addr)
	s.BaseContext = func(net.Listener) context.Context {
		return ctx
	}
	return s.ListenAndServe()
}

// Stop stop the server.
func (s *Server) Stop(ctx context.Context) error {
	LOG.Info("server stopping")
	return s.Shutdown(ctx)
}
