package main

import (
	"context"
	"flag"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/config"
	configLoader "github.com/go-kratos/gateway/config/config-loader"
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
	conf         string
	bind         string
	ctrlService  string
	discoveryDSN string
	adminAddr    string
	timeout      time.Duration
	idleTimeout  time.Duration
)

var (
	LOG = log.NewHelper(log.With(log.GetLogger(), "source", "main"))
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
		log.Fatalf("failed to create discovery: %v", err)
	}
	return impl
}

func main() {
	flag.Parse()
	go func() {
		LOG.Fatal(http.ListenAndServe(adminAddr, nil))
	}()

	clientFactory := client.NewFactory(makeDiscovery())
	p, err := proxy.New(clientFactory, middleware.Create)
	if err != nil {
		LOG.Fatalf("failed to new proxy: %v", err)
	}

	ctx := context.Background()
	if ctrlService != "" {
		LOG.Infof("setup control service to: %q", ctrlService)
		ctrlLoader := configLoader.New(ctrlService, conf)
		if err := ctrlLoader.Load(ctx); err != nil {
			LOG.Errorf("failed to do initial load from control service: %v, using local config instead", err)
		}
		go ctrlLoader.Run(ctx)
	}

	confLoader, err := config.NewFileLoader(conf)
	if err != nil {
		LOG.Fatalf("failed to create config file loader: %v", err)
	}
	defer confLoader.Close()
	bc, err := confLoader.Load(context.Background())
	if err != nil {
		LOG.Fatalf("failed to load config: %v", err)
	}

	if err := p.Update(bc); err != nil {
		LOG.Fatalf("failed to update service config: %v", err)
	}
	reloader := func() error {
		bc, err := confLoader.Load(context.Background())
		if err != nil {
			LOG.Errorf("failed to load config: %v", err)
			return err
		}
		if err := p.Update(bc); err != nil {
			LOG.Errorf("failed to update service config: %v", err)
			return err
		}
		LOG.Infof("config reloaded")
		return nil
	}
	confLoader.Watch(reloader)

	app := kratos.New(
		kratos.Name(bc.Name),
		kratos.Context(ctx),
		kratos.Server(
			server.New(p, bind, timeout, idleTimeout),
		),
	)
	if err := app.Run(); err != nil {
		LOG.Errorf("failed to run servers: %v", err)
	}
}
