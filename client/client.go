package client

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/core/v1"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/wrr"

	"golang.org/x/net/http2"
)

// Client is a proxy client.
type Client interface {
	Invoke(ctx context.Context, req *http.Request, opts ...CallOption) (*http.Response, error)
}

type clientImpl struct {
	selector selector.Selector
	nodes    map[string]*node
}

func (c *clientImpl) Invoke(ctx context.Context, req *http.Request, opts ...CallOption) (*http.Response, error) {
	callInfo := defaultCallInfo()
	for _, o := range opts {
		if err := o.before(&callInfo); err != nil {
			return nil, err
		}
	}
	selected, done, err := c.selector.Select(ctx, selector.WithFilter(callInfo.filters...))
	if err != nil {
		return nil, err
	}
	defer done(ctx, selector.DoneInfo{Err: err})
	node := c.nodes[selected.Address()]
	req.URL.Scheme = "http"
	req.URL.Host = selected.Address()
	req.Host = selected.Address()
	req.RequestURI = ""
	resp, err := node.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// NewFactory new a client factory.
func NewFactory() func(endpoint *config.Endpoint) (Client, error) {
	return func(endpoint *config.Endpoint) (Client, error) {
		c := &clientImpl{
			selector: wrr.New(),
			nodes:    make(map[string]*node),
		}
		var nodes []selector.Node
		for _, backend := range endpoint.Backends {
			client := &http.Client{
				Timeout: endpoint.Timeout.AsDuration(),
			}
			if endpoint.Protocol == config.Protocol_GRPC {
				client.Transport = &http2.Transport{
					// So http2.Transport doesn't complain the URL scheme isn't 'https'
					AllowHTTP: true,
					// Pretend we are dialing a TLS endpoint.
					// Note, we ignore the passed tls.Config
					DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
						return net.Dial(network, addr)
					},
				}
			}
			node := &node{
				protocol: endpoint.Protocol,
				address:  backend.Target,
				client:   client,
				weight:   backend.Weight,
			}
			nodes = append(nodes, node)
			c.nodes[backend.Target] = node
		}
		c.selector.Apply(nodes)
		return c, nil
	}
}
