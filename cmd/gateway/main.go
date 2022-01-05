package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"time"

	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/configloader"
	"github.com/go-kratos/gateway/configloader/ctrlloader"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/gateway/server"
	"github.com/hashicorp/consul/api"

	_ "github.com/go-kratos/gateway/middleware/color"
	_ "github.com/go-kratos/gateway/middleware/cors"
	_ "github.com/go-kratos/gateway/middleware/logging"
	_ "github.com/go-kratos/gateway/middleware/otel"
	_ "github.com/go-kratos/gateway/middleware/prometheus"
	_ "go.uber.org/automaxprocs"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"

	discoveryapi "git.bilibili.co/middleware/kratos/discovery"
)

var (
	conf        string
	ctrlService string
	bind        string
	timeout     time.Duration
	idleTimeout time.Duration
	// consul
	consulAddress    string
	consulToken      string
	consulDatacenter string
	// registry
	registryDSN string
	// debug
	adminAddr string
)

func init() {
	flag.StringVar(&conf, "conf", "config.yaml", "config path, eg: -conf config.yaml")
	flag.StringVar(&ctrlService, "ctrl.service", "", "control service host, eg: http://172.16.0.5:8000")
	flag.StringVar(&bind, "bind", ":8080", "server address, eg: 127.0.0.1:8080")
	flag.DurationVar(&timeout, "timeout", time.Second*15, "server timeout, eg: 15s")
	flag.DurationVar(&idleTimeout, "idleTimeout", time.Second*300, "server idleTimeout, eg: 300s")
	flag.StringVar(&consulAddress, "consul.address", "", "consul address, eg: 127.0.0.1:8500")
	flag.StringVar(&consulToken, "consul.token", "", "consul token, eg: xxx")
	flag.StringVar(&consulDatacenter, "consul.datacenter", "", "consul datacenter, eg: xxx")
	flag.StringVar(&adminAddr, "pprof", "0.0.0.0:7070", "admin addr, eg: 127.0.0.1:7070")
	flag.StringVar(&registryDSN, "registry.dsn", "", "registry dsn, eg: consul://127.0.0.1:7070?token=secret&datacenter=prod")
}

func makeRegistry() registry.Discovery {
	if registryDSN == "" {
		return nil
	}
	dsn, err := url.Parse(registryDSN)
	if err != nil {
		panic(err)
	}
	switch dsn.Scheme {
	case "consul":
		c := api.DefaultConfig()
		c.Address = dsn.Query().Get("address")
		c.Token = dsn.Query().Get("token")
		c.Datacenter = dsn.Query().Get("datacenter")
		client, err := api.NewClient(c)
		if err != nil {
			panic(err)
		}
		return consul.New(client)
	case "discovery":
		opts := []discoveryapi.Option{}
		region := dsn.Query().Get("region")
		if region != "" {
			opts = append(opts, discoveryapi.WithRegion(region))
		}
		zone := dsn.Query().Get("zone")
		if zone != "" {
			opts = append(opts, discoveryapi.WithZone(zone))
		}
		deployEnv := dsn.Query().Get("deployenv")
		if deployEnv != "" {
			opts = append(opts, discoveryapi.WithZone(deployEnv))
		}
		d := discoveryapi.New(opts...)
		return d
	default:
		panic(fmt.Errorf("unrecognized scheme: %s", dsn.Scheme))
	}
}

func main() {
	flag.Parse()
	logger := log.NewStdLogger(os.Stdout)
	log := log.NewHelper(logger)
	go func() {
		log.Fatal(http.ListenAndServe(adminAddr, nil))
	}()

	clientFactory := client.NewFactory(logger, makeRegistry())
	p, err := proxy.New(logger, clientFactory, middleware.Create)
	if err != nil {
		log.Fatalf("failed to new proxy: %v", err)
	}

	ctx := context.Background()
	if ctrlService != "" {
		log.Infof("setup control service to: %q", ctrlService)
		ctrlLoader := ctrlloader.New(ctrlService, conf)
		if err := ctrlLoader.Load(ctx); err != nil {
			log.Errorf("failed to do initial load from control service: %v, using local config instead", err)
		}
		go ctrlLoader.Run(context.TODO())
	}

	confLoader, err := configloader.NewFileLoader(conf)
	if err != nil {
		log.Fatalf("failed to create config file loader: %v", err)
	}
	defer confLoader.Close()
	bc, err := confLoader.Pull(context.Background())
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := p.Update(bc); err != nil {
		log.Fatalf("failed to update service config: %v", err)
	}
	reloader := func() {
		bc, err := confLoader.Pull(context.Background())
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
			return
		}
		if err := p.Update(bc); err != nil {
			log.Errorf("failed to update service config: %v", err)
			return
		}
		log.Infof("config reloaded")
	}
	confLoader.Watch(reloader)

	ctx = middleware.NewLoggingContext(ctx, logger)
	srv := server.New(logger, p, bind, timeout, idleTimeout)
	app := kratos.New(
		kratos.Name(bc.Name),
		kratos.Context(ctx),
		kratos.Server(srv),
	)
	if err := app.Run(); err != nil {
		log.Errorf("failed to run servers: %v", err)
	}
}
