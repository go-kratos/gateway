package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Server is a gateway server.
type Server struct {
	*http.Server

	log *log.Helper
}

// New new a gateway server.
func New(logger log.Logger, handler http.Handler, addr string, timeout time.Duration, idleTimeout time.Duration) *Server {
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
		log: log.NewHelper(logger),
	}
	return srv
}

// Start start the server.
func (s *Server) Start(ctx context.Context) error {
	s.log.Infof("server listening on %s", s.Addr)
	return s.ListenAndServe()
}

// Stop stop the server.
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("server stopping")
	return s.Shutdown(ctx)
}
