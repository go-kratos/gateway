package client

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/core/v1"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
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
	nodes    atomic.Value
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
	node := c.nodes.Load().(map[string]*node)[selected.Address()]
	req.URL.Scheme = "http"
	req.URL.Host = selected.Address()
	req.Host = selected.Address()
	req.RequestURI = ""
	log.Printf("client invoke %s", req.URL.String())
	resp, err := node.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// NewFactory new a client factory.
func NewFactory(consulClient *consul.Registry) func(endpoint *config.Endpoint) (Client, error) {
	return func(endpoint *config.Endpoint) (Client, error) {
		c := &clientImpl{
			selector: wrr.New(),
		}
		var nodes []selector.Node
		atomicNodes := make(map[string]*node, 0)
		for _, backend := range endpoint.Backends {
			target, err := parseTarget(backend.Target)
			if err != nil {
				return nil, err
			}
			if target.Scheme == "direct" {
				node := buildNode(backend.Target, endpoint.Protocol, backend.Weight, endpoint.Timeout.AsDuration())
				nodes = append(nodes, node)
				atomicNodes[backend.Target] = node
			} else if target.Scheme == "consul" {
				w, err := consulClient.Watch(context.Background(), target.Endpoint)
				if err != nil {
					return nil, err
				}
				go func() {
					// TODO: goroutine leak
					// only one backend configuration allowed when using service discovery
					for {
						services, err := w.Next()
						if err != nil && errors.Is(context.Canceled, err) {
							return
						}
						if len(services) != 0 {
							var nodes []selector.Node
							var atomicNodes map[string]*node = make(map[string]*node, 0)
							for _, ser := range services {
								addr, err := parseEndpoint(ser.Endpoints, strings.ToLower(endpoint.Protocol.String()), false)
								if err != nil {
									log.Printf("parse endpoint failed!err:=%v", err)
									continue
								}
								node := buildNode(addr, endpoint.Protocol, backend.Weight, endpoint.Timeout.AsDuration())
								nodes = append(nodes, node)
								atomicNodes[addr] = node
							}
							c.selector.Apply(nodes)
							c.nodes.Store(atomicNodes)
						}
					}
				}()
			} else if target.Scheme == "discovery" {
			}
		}
		c.selector.Apply(nodes)
		c.nodes.Store(atomicNodes)
		return c, nil
	}
}

func buildNode(addr string, protocol config.Protocol, weight *int64, timeout time.Duration) *node {
	client := &http.Client{
		Timeout: timeout,
	}
	if protocol == config.Protocol_GRPC {
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
		protocol: protocol,
		address:  addr,
		client:   client,
		weight:   weight,
	}
	return node
}

// Target is resolver target
type Target struct {
	Scheme    string
	Authority string
	Endpoint  string
}

func parseTarget(endpoint string) (*Target, error) {
	if !strings.Contains(endpoint, "://") {
		endpoint = "direct:///" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	target := &Target{Scheme: u.Scheme, Authority: u.Host}
	if len(u.Path) > 1 {
		target.Endpoint = u.Path[1:]
	}
	return target, nil
}

// parseEndpoint parses an Endpoint URL.
func parseEndpoint(endpoints []string, scheme string, isSecure bool) (string, error) {
	for _, e := range endpoints {
		u, err := url.Parse(e)
		if err != nil {
			return "", err
		}
		if u.Scheme == scheme {
			if IsSecure(u) == isSecure {
				return u.Host, nil
			}
		}
	}
	return "", nil
}

// IsSecure parses isSecure for Endpoint URL.
func IsSecure(u *url.URL) bool {
	ok, err := strconv.ParseBool(u.Query().Get("isSecure"))
	if err != nil {
		return false
	}
	return ok
}
