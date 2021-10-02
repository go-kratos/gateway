package main

import (
	"context"
	"flag"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/gateway"
	"github.com/go-kratos/gateway/router/mux"

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

	router := mux.NewRouter()

	// TODO buildRoute

	if err := gateway.Run(context.Background(), router, bc.Gateways...); err != nil {
		panic(err)
	}

}
