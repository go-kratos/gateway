package server

import (
	"context"
	"log"
	"net/http"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Run run a gateway server.
func Run(ctx context.Context, handler http.Handler, cs []*config.Gateway) error {
	done := make(chan error)
	for _, c := range cs {
		srv := &http.Server{
			Addr: c.Address,
			Handler: h2c.NewHandler(handler, &http2.Server{
				IdleTimeout: time.Second * 120,
			}),
			ReadTimeout:       time.Second * 5,
			WriteTimeout:      time.Second * 5,
			ReadHeaderTimeout: time.Second * 5,
			IdleTimeout:       time.Second * 120,
		}
		log.Printf("gateway listening on %s\n", c.Address)
		go func() {
			if c.TlsConfig != nil {
				done <- srv.ListenAndServeTLS(c.TlsConfig.PrivateKey, c.TlsConfig.PublicKey)
			} else {
				done <- srv.ListenAndServe()
			}
		}()
		go func() {
			<-ctx.Done()
			done <- srv.Shutdown(context.Background())
		}()
	}
	return <-done
}
