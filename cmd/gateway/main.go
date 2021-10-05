package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/middleware/cors"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/gateway/server"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
)

var flagconf string

func init() {
	flag.StringVar(&flagconf, "conf", "config.yaml", "config path, eg: -conf config.yaml")
}

func middlewares(ms []*configv1.Middleware, handler http.Handler) (http.Handler, error) {
	for _, m := range ms {
		switch m.Name {
		case cors.Name:
			handler = cors.Middleware(m)(handler)
		default:
			return nil, fmt.Errorf("not found middleware: %s", m.Name)
		}
	}
	return handler, nil
}

func main() {
	flag.Parse()
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	if err := c.Load(); err != nil {
		panic(err)
	}

	bc := new(configv1.Bootstrap)
	if err := c.Scan(bc); err != nil {
		panic(err)
	}

	p, err := proxy.New(client.NewFactory(), middlewares)
	if err != nil {
		panic(err)
	}
	if err := p.Update(bc.Services); err != nil {
		panic(err)
	}
	if err := server.Run(context.Background(), p, bc.Gateways); err != nil {
		panic(err)
	}
}
