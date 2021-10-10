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
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	if err := c.Load(); err != nil {
		panic(err)
	}
	log := log.NewHelper(log.NewStdLogger(os.Stdout))

	bc := new(configv1.Bootstrap)
	if err := c.Scan(bc); err != nil {
		log.Fatalf("config scan Bootstrap failed!err:=%v", err)
	}

	p, err := proxy.New(client.NewFactory(), middlewares)
	if err != nil {
		log.Fatalf("new proxy failed!err:=%v", err)
	}
	if err := p.Update(bc.Services); err != nil {
		log.Fatalf("update services failed!err:=%v", err)
	}
	c.Watch("services", func(_ string, v config.Value) {
		vals, err := v.Slice()
		if err != nil {
			log.Errorf("watch config change failed!err:=%v", err)
			return
		}
		var services []*configv1.Service
		for _, val := range vals {
			var ser configv1.Service
			err = val.Scan(&ser)
			if err != nil {
				log.Errorf("watch config change failed!err:=%v", err)
				return
			}
			services = append(services, &ser)
		}

		err = p.Update(services)
		if err != nil {
			log.Errorf("update service config failed!err:=%v", err)
			return
		}
	})
	if err := server.Run(context.Background(), p, bc.Gateways); err != nil {
		log.Errorf("server run failed!err:=%v", err)
	}
}
