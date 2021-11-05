package main

import (
	"context"
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/gateway/server"
	"github.com/hashicorp/consul/api"

	_ "github.com/go-kratos/gateway/middleware/cors"
	_ "github.com/go-kratos/gateway/middleware/dyeing"
	_ "github.com/go-kratos/gateway/middleware/logging"
	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	conf        string
	bind        string
	timeout     time.Duration
	idleTimeout time.Duration
	// consul
	consulAddress    string
	consulToken      string
	consulDatacenter string
	// debug
	pprofAddr string
)

func init() {
	flag.StringVar(&conf, "conf", "config.yaml", "config path, eg: -conf config.yaml")
	flag.StringVar(&bind, "bind", ":8080", "server address, eg: 127.0.0.1:8080")
	flag.DurationVar(&timeout, "timeout", time.Second*15, "server timeout, eg: 15s")
	flag.DurationVar(&idleTimeout, "idleTimeout", time.Second*300, "server idleTimeout, eg: 300s")
	flag.StringVar(&consulAddress, "consul.address", "", "consul address, eg: 127.0.0.1:8500")
	flag.StringVar(&consulToken, "consul.token", "", "consul token, eg: xxx")
	flag.StringVar(&consulDatacenter, "consul.datacenter", "", "consul datacenter, eg: xxx")
	flag.StringVar(&pprofAddr, "pprof", "0.0.0.0:7070", "pprof addr, eg: 127.0.0.1:7070")
}

func registry() *consul.Registry {
	if consulAddress != "" {
		c := api.DefaultConfig()
		c.Address = consulAddress
		c.Token = consulToken
		c.Datacenter = consulDatacenter
		client, err := api.NewClient(c)
		if err != nil {
			panic(err)
		}
		return consul.New(client)
	}
	return nil
}

func main() {
	flag.Parse()
	logger := log.NewStdLogger(os.Stdout)
	log := log.NewHelper(logger)
	go func() {
		log.Fatal(http.ListenAndServe(pprofAddr, nil))
	}()
	c := config.New(
		config.WithLogger(logger),
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
	ctx := context.Background()
	ctx = middleware.NewLoggingContext(ctx, logger)
	clientFactory := client.NewFactory(logger, registry())
	p, err := proxy.New(ctx, logger, clientFactory, middleware.Create)
	if err != nil {
		log.Fatalf("failed to new proxy: %v", err)
	}
	if err := p.Update(bc); err != nil {
		log.Fatalf("failed to update service config: %v", err)
	}
	srv := server.New(logger, p, bind, timeout, idleTimeout)
	app := kratos.New(
		kratos.Name(bc.Name),
		kratos.Server(srv),
	)
	if err := app.Run(); err != nil {
		log.Errorf("failed to run servers: %v", err)
	}
}
