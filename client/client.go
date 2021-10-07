package client

import (
	"context"
	"fmt"
	"log"
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
)

// Client is a proxy client.
type Client interface {
	Invoke(ctx context.Context, req *http.Request, opts ...CallOption) (*http.Response, error)
}

type clientImpl struct{}

func (c *clientImpl) Invoke(ctx context.Context, req *http.Request, opts ...CallOption) (*http.Response, error) {
	log.Printf("invoke [%s] %s\n", req.Method, req.URL.Path)
	return nil, fmt.Errorf("not implemented")
}

// NewFactory new a client factory.
func NewFactory() func(c *config.Endpoint) (Client, error) {
	return func(c *config.Endpoint) (Client, error) {
		// TODO new a proxy client
		return &clientImpl{}, nil
	}
}
