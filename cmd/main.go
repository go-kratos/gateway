package main

import (
	"flag"
	"fmt"

	"github.com/go-kratos/gateway/router"
	"github.com/go-kratos/gateway/server"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
)

var flagconf string

func init() {
	flag.StringVar(&flagconf, "conf", "config.yaml", "config path, eg: -conf config.yaml")
}

type ruleSource struct {
	cfg config.Config
}

func (source *ruleSource) Watch(f func(rule *router.RouteRule)) error {
	return source.cfg.Watch("routeRule", func(k string, v config.Value) {
		var routeRule router.RouteRule
		err := v.Scan(&routeRule)
		if err == nil {
			f(&routeRule)
		} else {
			fmt.Println("watch key routeRule failed!")
		}
	})
}

func (source *ruleSource) Load() (*router.RouteRule, error) {
	var routeRule router.RouteRule

	err := source.cfg.Value("routeRule").Scan(&routeRule)
	if err != nil {
		return nil, err
	}
	fmt.Printf("%+v", routeRule)
	return &routeRule, nil
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
	source := &ruleSource{
		cfg: c,
	}
	r, err := router.New(source)
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
