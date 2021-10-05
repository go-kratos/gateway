package client

import (
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
)

// Client is a proxy client.
type Client interface {
	Invoke(w http.ResponseWriter, req *http.Request)
}

// NewFactory new a client factory.
func NewFactory() func(service *config.Service) (Client, error) {
	return func(service *config.Service) (Client, error) {
		// TODO new a proxy client
		return nil, nil
	}
}
