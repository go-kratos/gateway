package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"

	"github.com/go-kratos/kratos/v2/log"
)

// Server is gateway server
type Server struct {
	httpSrv *http.Server

	opts    *options
	address string
	handler http.Handler
}

// New creates new server
func New(address string, handler http.Handler, opts ...Option) (*Server, error) {
	o := &options{
		log:     log.NewHelper(log.DefaultLogger),
		network: "tcp",
	}
	for _, opt := range opts {
		opt(o)
	}
	if o.endpoint == nil {
		addr, err := parseHostPort(address)
		if err != nil {
			return nil, err
		}
		o.endpoint = newEndpoint("http", addr, o.tlsConf != nil)
	}

	s := &Server{
		opts:    o,
		address: address,
		handler: handler,
	}
	s.httpSrv = &http.Server{
		TLSConfig: o.tlsConf,
		Handler:   s,
	}
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

// Start start the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	s.httpSrv.BaseContext = func(net.Listener) context.Context {
		return ctx
	}
	lis, err := net.Listen(s.opts.network, s.address)
	if err != nil {
		return err
	}

	s.opts.log.Infof("[HTTP] server listening on: %s", lis.Addr().String())

	if s.httpSrv.TLSConfig != nil {
		err = s.httpSrv.ServeTLS(lis, "", "")
	} else {
		err = s.httpSrv.Serve(lis)
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Stop stop the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	s.opts.log.Info("[HTTP] server stopping")
	return s.httpSrv.Shutdown(ctx)
}

// Endpoint return server endpoint
func (s *Server) Endpoint() (*url.URL, error) {
	return s.opts.endpoint, nil
}
