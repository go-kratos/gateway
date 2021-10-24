package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/endpoint"
	"github.com/go-kratos/gateway/endpoint/cors"
	"github.com/go-kratos/gateway/endpoint/dyeing"
	"github.com/go-kratos/gateway/endpoint/retry"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/gateway/server"
	"github.com/hashicorp/consul/api"

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
	// service
	serviceName string
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
	flag.StringVar(&serviceName, "service.name", "gateway", "service name, eg: gateway")
	flag.StringVar(&consulAddress, "consul.address", "", "consul address, eg: 127.0.0.1:8500")
	flag.StringVar(&consulToken, "consul.token", "", "consul token, eg: xxx")
	flag.StringVar(&consulDatacenter, "consul.datacenter", "", "consul datacenter, eg: xxx")
	flag.StringVar(&pprofAddr, "pprof", "", "pprof addr, eg: localhost:8088")
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

func middlewareFactory(c *configv1.Middleware) (endpoint.Middleware, error) {
	switch c.Name {
	case cors.Name:
		return cors.Middleware(c)
	case dyeing.Name:
		return dyeing.Middleware(c)
	case retry.Name:
		return retry.Middleware(c)
	default:
		return nil, fmt.Errorf("not found middleware: %s", c.Name)
	}
}

func main() {
	flag.Parse()
	logger := log.NewStdLogger(os.Stdout)
	log := log.NewHelper(logger)
	if pprofAddr != "" {
		go pprofServer(log)
	}

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
	clientFactory := client.NewFactory(logger, registry())
	p, err := proxy.New(logger, clientFactory, middlewareFactory)
	if err != nil {
		log.Fatalf("failed to new proxy: %v", err)
	}
	if err := p.Update(bc); err != nil {
		log.Fatalf("failed to update service config: %v", err)
	}
	srv := server.New(logger, p, bind, timeout, idleTimeout)
	app := kratos.New(
		kratos.Name(serviceName),
		kratos.Server(srv),
	)
	if err := app.Run(); err != nil {
		log.Errorf("failed to run servers: %v", err)
	}
}

func pprofServer(log *log.Helper) {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	log.Infof("pprof server start listening on: %s", pprofAddr)
	err := http.ListenAndServe(pprofAddr, mux)
	if err != nil {
		log.Errorf("failed to run pprof server: %v", err)
	}
}
