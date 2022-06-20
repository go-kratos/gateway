package server

import (
	"context"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var (
	readHeaderTimeout = time.Second * 10
	readTimeout       = time.Second * 15
	writeTimeout      = time.Second * 15
	idleTimeout       = time.Second * 120
)

func init() {
	var err error
	if v := os.Getenv("PROXY_READ_HEADER_TIMEOUT"); v != "" {
		if readHeaderTimeout, err = time.ParseDuration(v); err != nil {
			panic(err)
		}
	}
	if v := os.Getenv("PROXY_READ_TIMEOUT"); v != "" {
		if readTimeout, err = time.ParseDuration(v); err != nil {
			panic(err)
		}
	}
	if v := os.Getenv("PROXY_WRITE_TIMEOUT"); v != "" {
		if writeTimeout, err = time.ParseDuration(v); err != nil {
			panic(err)
		}
	}
	if v := os.Getenv("PROXY_IDLE_TIMEOUT"); v != "" {
		if idleTimeout, err = time.ParseDuration(v); err != nil {
			panic(err)
		}
	}
}

// ProxyServer is a proxy server.
type ProxyServer struct {
	*http.Server
}

// NewProxy new a gateway server.
func NewProxy(handler http.Handler, addr string) *ProxyServer {
	return &ProxyServer{
		Server: &http.Server{
			Addr: addr,
			Handler: h2c.NewHandler(handler, &http2.Server{
				IdleTimeout:          idleTimeout,
				MaxConcurrentStreams: math.MaxUint32,
			}),
			ReadTimeout:       readTimeout,
			ReadHeaderTimeout: readHeaderTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
	}
}

// Start the server.
func (s *ProxyServer) Start(ctx context.Context) error {
	log.Infof("proxy listening on %s", s.Addr)
	return s.ListenAndServe()
}

// Stop the server.
func (s *ProxyServer) Stop(ctx context.Context) error {
	log.Info("proxy stopping")
	return s.Shutdown(ctx)
}
