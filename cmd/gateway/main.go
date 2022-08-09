package main

import (
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/config"
	configLoader "github.com/go-kratos/gateway/config/config-loader"
	"github.com/go-kratos/gateway/discovery"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/gateway/proxy/debug"
	"github.com/go-kratos/gateway/server"

	_ "net/http/pprof"

	_ "github.com/go-kratos/gateway/discovery/consul"
	_ "github.com/go-kratos/gateway/discovery/nacos"
	_ "github.com/go-kratos/gateway/middleware/bbr"
	"github.com/go-kratos/gateway/middleware/circuitbreaker"
	_ "github.com/go-kratos/gateway/middleware/cors"
	_ "github.com/go-kratos/gateway/middleware/logging"
	_ "github.com/go-kratos/gateway/middleware/rewrite"
	_ "github.com/go-kratos/gateway/middleware/tracing"
	_ "github.com/go-kratos/gateway/middleware/transcoder"
	_ "go.uber.org/automaxprocs"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
)

var (
	ctrlName     string
	ctrlService  string
	discoveryDSN string
	proxyAddr    string
	proxyConfig  string
	withDebug    bool
)

func init() {
	flag.BoolVar(&withDebug, "debug", false, "enable debug handlers")
	flag.StringVar(&proxyAddr, "addr", ":8080", "proxy address, eg: -addr 0.0.0.0:8080")
	flag.StringVar(&proxyConfig, "conf", "config.yaml", "config path, eg: -conf config.yaml")
	flag.StringVar(&ctrlName, "ctrl.name", os.Getenv("ADVERTISE_NAME"), "control gateway name, eg: gateway")
	flag.StringVar(&ctrlService, "ctrl.service", "", "control service host, eg: http://127.0.0.1:8000")
	flag.StringVar(&discoveryDSN, "discovery.dsn", "", "discovery dsn, eg: consul://127.0.0.1:7070?token=secret&datacenter=prod")
}

func makeDiscovery() registry.Discovery {
	if discoveryDSN == "" {
		return nil
	}
	d, err := discovery.Create(discoveryDSN)
	if err != nil {
		log.Fatalf("failed to create discovery: %v", err)
	}
	return d
}

func main() {
	flag.Parse()

	clientFactory := client.NewFactory(makeDiscovery())
	p, err := proxy.New(clientFactory, middleware.Create)
	if err != nil {
		log.Fatalf("failed to new proxy: %v", err)
	}
	circuitbreaker.Init(clientFactory)

	ctx := context.Background()
	var ctrlLoader *configLoader.CtrlConfigLoader
	if ctrlService != "" {
		log.Infof("setup control service to: %q", ctrlService)
		ctrlLoader = configLoader.New(ctrlName, ctrlService, proxyConfig)
		if err := ctrlLoader.Load(ctx); err != nil {
			log.Errorf("failed to do initial load from control service: %v, using local config instead", err)
		}
		go ctrlLoader.Run(ctx)
	}

	confLoader, err := config.NewFileLoader(proxyConfig)
	if err != nil {
		log.Fatalf("failed to create config file loader: %v", err)
	}
	defer confLoader.Close()
	bc, err := confLoader.Load(context.Background())
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := p.Update(bc); err != nil {
		log.Fatalf("failed to update service config: %v", err)
	}
	reloader := func() error {
		bc, err := confLoader.Load(context.Background())
		if err != nil {
			log.Errorf("failed to load config: %v", err)
			return err
		}
		if err := p.Update(bc); err != nil {
			log.Errorf("failed to update service config: %v", err)
			return err
		}
		log.Infof("config reloaded")
		return nil
	}
	confLoader.Watch(reloader)

	var serverHandler http.Handler = p
	if withDebug {
		debug.Register("proxy", p)
		debug.Register("config", confLoader)
		if ctrlLoader != nil {
			debug.Register("ctrl", ctrlLoader)
		}
		serverHandler = debug.MashupWithDebugHandler(p)
	}
	app := kratos.New(
		kratos.Name(bc.Name),
		kratos.Context(ctx),
		kratos.Server(
			server.NewProxy(serverHandler, proxyAddr),
		),
	)
	if err := app.Run(); err != nil {
		log.Errorf("failed to run servers: %v", err)
	}
}
