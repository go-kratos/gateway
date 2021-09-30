package main

import (
	"flag"

	"github.com/go-kratos/gateway/server"
	"github.com/go-kratos/gateway/service"
	"github.com/go-kratos/gateway/source"

	"github.com/go-kratos/kratos/v2"
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

	r, err := service.New(source.NewRule(c))
	if err != nil {
		panic(err)
	}
	server, err := server.New(":8080", r)
	if err != nil {
		panic(err)
	}
	app := kratos.New(kratos.Server(server))
	err = app.Run()
	if err != nil {
		panic(err)
	}
}
