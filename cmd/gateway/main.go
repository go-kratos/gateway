package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/configloader"
	"github.com/go-kratos/gateway/configloader/ctrlloader"
	"github.com/go-kratos/gateway/discovery"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/gateway/server"

	_ "github.com/go-kratos/gateway/discovery/consul"
	_ "github.com/go-kratos/gateway/middleware/color"
	_ "github.com/go-kratos/gateway/middleware/cors"
	_ "github.com/go-kratos/gateway/middleware/logging"
	_ "github.com/go-kratos/gateway/middleware/otel"
	_ "github.com/go-kratos/gateway/middleware/prometheus"
	_ "go.uber.org/automaxprocs"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
)

var (
	conf        string
	ctrlService string
	bind        string
	timeout     time.Duration
	idleTimeout time.Duration
	// discovery
	discoveryDSN string
	// debug
	adminAddr string
)

func init() {
	flag.StringVar(&conf, "conf", "config.yaml", "config path, eg: -conf config.yaml")
	flag.StringVar(&ctrlService, "ctrl.service", "", "control service host, eg: http://172.16.0.5:8000")
	flag.StringVar(&bind, "bind", ":8080", "server address, eg: 127.0.0.1:8080")
	flag.DurationVar(&timeout, "timeout", time.Second*15, "server timeout, eg: 15s")
	flag.DurationVar(&idleTimeout, "idleTimeout", time.Second*300, "server idleTimeout, eg: 300s")
	flag.StringVar(&adminAddr, "pprof", "0.0.0.0:7070", "admin addr, eg: 127.0.0.1:7070")
	flag.StringVar(&discoveryDSN, "discovery.dsn", "", "discovery dsn, eg: consul://127.0.0.1:7070?token=secret&datacenter=prod")
}

func makeDiscovery() registry.Discovery {
	if discoveryDSN == "" {
		return nil
	}

	impl, err := discovery.Create(discoveryDSN)
	if err != nil {
		panic(fmt.Errorf("failed to create discovery: %v", err))
	}
	return impl
}

func main() {
	flag.Parse()
	logger := log.NewStdLogger(os.Stdout)
	log := log.NewHelper(logger)
	go func() {
		log.Fatal(http.ListenAndServe(adminAddr, nil))
	}()

	clientFactory := client.NewFactory(logger, makeDiscovery())
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
