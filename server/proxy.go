package server

import (
	"context"
	"net/http"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	// LOG .
	LOG = log.NewHelper(log.With(log.GetLogger(), "source", "server"))
)

// ProxyServer is a proxy server.
type ProxyServer struct {
	*http.Server
}

// NewProxy new a gateway server.
func NewProxy(handler http.Handler, c *config.Gateway) *ProxyServer {
	if c.Address == "" {
		c.Address = ":8080"
	}
	if c.ReadTimeout == nil {
		c.ReadTimeout = durationpb.New(time.Second * 15)
	}
	if c.WriteTimeout == nil {
		c.WriteTimeout = durationpb.New(time.Second * 15)
	}
	if c.IdleTimeout == nil {
		c.IdleTimeout = durationpb.New(time.Second * 300)
	}
	return &ProxyServer{
		Server: &http.Server{
			Addr: c.Address,
			Handler: h2c.NewHandler(handler, &http2.Server{
				IdleTimeout: c.IdleTimeout.AsDuration(),
			}),
			ReadTimeout:       c.ReadTimeout.AsDuration(),
			ReadHeaderTimeout: c.ReadTimeout.AsDuration(),
			WriteTimeout:      c.WriteTimeout.AsDuration(),
			IdleTimeout:       c.IdleTimeout.AsDuration(),
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
