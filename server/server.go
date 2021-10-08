package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/kratos/v2/selector"
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
			ConnContext: func(ctx context.Context, c net.Conn) context.Context {
				return proxy.NewContext(ctx, &proxy.RequestOptions{
					Filters: []selector.Filter{},
				})
			},
			ReadTimeout:       c.Timeout.AsDuration(),
			WriteTimeout:      c.Timeout.AsDuration(),
			ReadHeaderTimeout: c.Timeout.AsDuration(),
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
	if c.Timeout == nil {
		c.Timeout = durationpb.New(time.Second * 15)
	}
	if c.IdleTimeout == nil {
		c.IdleTimeout = durationpb.New(time.Second * 300)
	}
	return nil
}
