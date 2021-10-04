package main

import (
	"context"
	"flag"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/gateway/server"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
)

var flagconf string

func init() {
	flag.StringVar(&flagconf, "conf", "config.yaml", "config path, eg: -conf config.yaml")
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

	// TODO add client manager
	p, err := proxy.New(nil)
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
