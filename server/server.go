package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Run run a gateway server.
func Run(ctx context.Context, handler http.Handler, addr string, timeout time.Duration, idleTimeout time.Duration) error {
	done := make(chan error)
	srv := &http.Server{
		Addr: addr,
		Handler: h2c.NewHandler(handler, &http2.Server{
			IdleTimeout: idleTimeout,
		}),
		ReadTimeout:       timeout,
		ReadHeaderTimeout: timeout,
		WriteTimeout:      timeout,
		IdleTimeout:       idleTimeout,
	}
	log.Printf("gateway listening on %s\n", addr)
	go func() {
		done <- srv.ListenAndServe()
	}()
	go func() {
		<-ctx.Done()
		done <- srv.Shutdown(context.Background())
	}()
	return <-done
}
