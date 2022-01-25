package server

import (
	"context"
	"net/http"
)

// AdminServer is a admin server.
type AdminServer struct {
	*http.Server
}

// NewAdmin new a admin server.
func NewAdmin(addr string) *AdminServer {
	return &AdminServer{
		Server: &http.Server{
			Addr:    addr,
			Handler: http.DefaultServeMux,
		},
	}
}

// Start start the server.
func (s *AdminServer) Start(ctx context.Context) error {
	LOG.Infof("admin server listening on %s", s.Addr)
	return s.ListenAndServe()
}

// Stop stop the server.
func (s *AdminServer) Stop(ctx context.Context) error {
	LOG.Info("admin server stopping")
	return s.Shutdown(ctx)
}
