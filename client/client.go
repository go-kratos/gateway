package client

import (
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
)

// Client is a proxy client.
type Client interface {
	Invoke(w http.ResponseWriter, req *http.Request)
}

// NewClient new a proxy client.
func NewClient(c *config.Service) (Client, error) {
	return nil, nil
}
