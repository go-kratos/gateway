package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	core "github.com/go-kratos/gateway/api/gateway/core/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/middleware/cors"
	"github.com/go-kratos/gateway/middleware/dyeing"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/gateway/server"
	"github.com/hashicorp/consul/api"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
)

var flagconf string

func init() {
	flag.StringVar(&flagconf, "conf", "config.yaml", "config path, eg: -conf config.yaml")
}

func middlewares(c *core.Middleware) (middleware.Middleware, error) {
	switch c.Name {
	case cors.Name:
		return cors.Middleware(c)
	case dyeing.Name:
		return dyeing.Middleware(c)
	default:
		return nil, fmt.Errorf("not found middleware: %s", c.Name)
	}
}

func main() {
	flag.Parse()
	log := log.NewHelper(log.NewStdLogger(os.Stdout))
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	if err := c.Load(); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	bc := new(core.Bootstrap)
	if err := c.Scan(bc); err != nil {
		log.Fatalf("failed to scan config: %v", err)
	}
	var r *consul.Registry
	if bc.Consul != nil {
		consulCfg := api.DefaultConfig()
		consulCfg.Address = bc.Consul.Address
		if bc.Consul.Token != "" {
			consulCfg.Token = bc.Consul.Token
		}
		if bc.Consul.Datacenter != "" {
			consulCfg.Datacenter = bc.Consul.Datacenter
		}
		consulCli, err := api.NewClient(api.DefaultConfig())
		if err != nil {
			log.Fatalf("failed to new consul: %v", err)
		}
		r = consul.New(consulCli)
	}

	p, err := proxy.New(client.NewFactory(r), middlewares)
	if err != nil {
		log.Fatalf("failed to new proxy: %v", err)
	}
	if err := p.Update(bc.Services); err != nil {
		log.Fatalf("failed to update service config: %v", err)
	}
	c.Watch("services", func(k string, v config.Value) {
		vals, err := v.Slice()
		if err != nil {
			log.Errorf("failed to watch config change: %v", err)
			return
		}
		var services []*core.Service
		for _, val := range vals {
			var sc core.Service
			if err = val.Scan(&sc); err != nil {
				log.Errorf("failed to watch config change: %v", err)
				return
			}
			services = append(services, &sc)
		}
		if err = p.Update(services); err != nil {
			log.Errorf("failed to update service config: %v", err)
		}
	})
	if err := server.Run(context.Background(), p, bc.Gateways); err != nil {
		log.Errorf("failed to run servers: %v", err)
	}
}
