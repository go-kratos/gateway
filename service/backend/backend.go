package backend

import (
	"net/http"

	"github.com/go-kratos/gateway/api"

	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/wrr"
)

type Backend struct {
	*api.Backend
}

func (b *Backend) Build() *Client {
	s := wrr.New()
	var nodes []selector.Node
	for _, target := range b.Target {
		nodes = append(nodes, &node{address: target})
	}
	s.Apply(nodes)
	return &Client{
		client:   &http.Client{},
		selector: s,
		scheme:   b.Scheme,
	}
}
