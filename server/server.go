package server

import (
	"context"
	"log"
	"net/http"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Run run a gateway server.
func Run(ctx context.Context, handler http.Handler, cs []*config.Gateway) error {
	done := make(chan error)
	for _, c := range cs {
		if err := validateConfig(c); err != nil {
			return err
		}
		srv := &http.Server{
			Addr: c.Address,
			Handler: h2c.NewHandler(handler, &http2.Server{
				IdleTimeout: c.IdleTimeout.AsDuration(),
			}),
			ReadTimeout:       c.ReadTimeout.AsDuration(),
			ReadHeaderTimeout: c.ReadHeaderTimeout.AsDuration(),
			WriteTimeout:      c.WriteTimeout.AsDuration(),
			IdleTimeout:       c.IdleTimeout.AsDuration(),
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

func validateConfig(c *config.Gateway) error {
	if c.ReadTimeout == nil {
		c.ReadTimeout = durationpb.New(time.Second * 15)
	}
	if c.ReadHeaderTimeout == nil {
		c.ReadHeaderTimeout = durationpb.New(time.Second * 15)
	}
	if c.WriteTimeout == nil {
		c.WriteTimeout = durationpb.New(time.Second * 15)
	}
	if c.IdleTimeout == nil {
		c.IdleTimeout = durationpb.New(time.Second * 300)
	}
	return nil
}
