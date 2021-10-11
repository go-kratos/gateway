package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
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

var (
	conf        string
	bind        string
	timeout     time.Duration
	idleTimeout time.Duration
)

func init() {
	flag.StringVar(&conf, "conf", "config.yaml", "config path, eg: -conf config.yaml")
	flag.StringVar(&bind, "bind", ":8080", "server address, eg: 127.0.0.1:8080")
	flag.DurationVar(&timeout, "timeout", time.Second*15, "server timeout, eg: 15s")
	flag.DurationVar(&idleTimeout, "idleTimeout", time.Second*300, "server idleTimeout, eg: 300s")
}

func middlewares(c *configv1.Middleware) (middleware.Middleware, error) {
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
			file.NewSource(conf),
		),
	)
	if err := c.Load(); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	bc := new(configv1.Gateway)
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
	if err := p.Update(bc); err != nil {
		log.Fatalf("failed to update service config: %v", err)
	}
	if err := server.Run(context.Background(), p, bind, timeout, idleTimeout); err != nil {
		log.Errorf("failed to run servers: %v", err)
	}
}
