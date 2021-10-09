package client

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"golang.org/x/net/http2"

	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/wrr"
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
	scheme := "http"
	req.URL.Scheme = scheme
	req.URL.Host = selected.Address()
	req.Host = selected.Address()
	req.RequestURI = ""
	// url := fmt.Sprintf("%s://%s%s", scheme, selected.Address(), req.URL.Path)
	/*header := req.Header
	req, err = http.NewRequest(req.Method, url, req.Body)
	if err != nil {
		return nil, err
	}
	for k, values := range header {
		req.Header[k] = values
	}*/
	log.Printf("invoke [%s] %s\n", req.Method, req.URL.String())

	resp, err := node.client.Do(req)
	if err != nil {
		log.Printf("invoke error: %s\n", err.Error())
		return nil, err
	}
	return resp, nil
}

// NewFactory new a client factory.
func NewFactory() func(protocol config.Protocol, backends []*config.Backend) (Client, error) {
	return func(protocol config.Protocol, backends []*config.Backend) (Client, error) {
		s := wrr.New()
		c := &clientImpl{
			selector: s,
			nodes:    make(map[string]*node),
		}
		var nodes []selector.Node
		for _, backend := range backends {
			var client *http.Client
			if protocol == config.Protocol_GRPC {
				client = &http.Client{
					Timeout: time.Second * 15,
					Transport: &http2.Transport{
						// So http2.Transport doesn't complain the URL scheme isn't 'https'
						AllowHTTP: true,
						// Pretend we are dialing a TLS endpoint.
						// Note, we ignore the passed tls.Config
						DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
							return net.Dial(network, addr)
						},
					},
				}
			} else {
				client = &http.Client{
					Timeout: time.Second * 15,
				}
			}
			node := &node{
				address:  backend.Target,
				client:   client,
				protocol: protocol,
				weight:   backend.Weight,
			}
			nodes = append(nodes, node)
			c.nodes[backend.Target] = node
		}
		s.Apply(nodes)

		return c, nil
	}
}
