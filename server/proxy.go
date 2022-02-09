package server

import (
	"context"
	"math"
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

	_defaultAddress           = ":8080"
	_defaultReadHeaderTimeout = time.Second * 10
	_defaultReadTimeout       = time.Second * 15
	_defaultWriteTimeout      = time.Second * 15
	_defaultIdleTimeout       = time.Second * 300
)

// ProxyServer is a proxy server.
type ProxyServer struct {
	*http.Server
}

// NewProxy new a gateway server.
func NewProxy(handler http.Handler, c *config.Gateway) *ProxyServer {
	if c.Address == "" {
		c.Address = _defaultAddress
	}
	if c.ReadHeaderTimeout == nil {
		c.ReadHeaderTimeout = durationpb.New(_defaultReadHeaderTimeout)
	}
	if c.ReadTimeout == nil {
		c.ReadTimeout = durationpb.New(_defaultReadTimeout)
	}
	if c.WriteTimeout == nil {
		c.WriteTimeout = durationpb.New(_defaultWriteTimeout)
	}
	if c.IdleTimeout == nil {
		c.IdleTimeout = durationpb.New(_defaultIdleTimeout)
	}
	return &ProxyServer{
		Server: &http.Server{
			Addr: c.Address,
			Handler: h2c.NewHandler(handler, &http2.Server{
				IdleTimeout:          c.IdleTimeout.AsDuration(),
				MaxConcurrentStreams: math.MaxUint32,
			}),
			ReadTimeout:       c.ReadTimeout.AsDuration(),
			ReadHeaderTimeout: c.ReadHeaderTimeout.AsDuration(),
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
