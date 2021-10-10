package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/middleware/cors"
	"github.com/go-kratos/gateway/middleware/dyeing"
	"github.com/go-kratos/gateway/proxy"
	"github.com/go-kratos/gateway/server"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
)

var flagconf string

func init() {
	flag.StringVar(&flagconf, "conf", "config.yaml", "config path, eg: -conf config.yaml")
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
			file.NewSource(flagconf),
		),
	)
	if err := c.Load(); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	bc := new(configv1.Bootstrap)
	if err := c.Scan(bc); err != nil {
		log.Fatalf("failed to scan config: %v", err)
	}
	p, err := proxy.New(client.NewFactory(), middlewares)
	if err != nil {
		log.Fatalf("failed to new proxy: %v", err)
	}
	if err := p.Update(bc.Services); err != nil {
		log.Fatalf("failed to update service config: %v", err)
	}
	c.Watch("services", func(_ string, v config.Value) {
		vals, err := v.Slice()
		if err != nil {
			log.Errorf("failed to watch config change: %v", err)
			return
		}
		var services []*configv1.Service
		for _, val := range vals {
			var sc configv1.Service
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
