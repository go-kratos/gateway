package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync/atomic"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/endpoint"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"

	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/wrr"
)

// Client is a proxy client.
type Client interface {
	Invoke(ctx context.Context, req *http.Request) (*http.Response, error)
}

type clientImpl struct {
	selector selector.Selector
	nodes    atomic.Value

	attempts        uint32
	allowTriedNodes bool
	conditions      []string
}

func (c *clientImpl) Invoke(ctx context.Context, req *http.Request) (*http.Response, error) {
	opts, _ := endpoint.FromContext(ctx)
	selected, done, err := c.selector.Select(ctx, selector.WithFilter(opts.Filters...))
	if err != nil {
		return nil, err
	}
	defer done(ctx, selector.DoneInfo{Err: err})
	node := c.nodes.Load().(map[string]*node)[selected.Address()]
	req.URL.Scheme = "http"
	req.URL.Host = selected.Address()
	req.RequestURI = ""
	req = req.WithContext(ctx)

	if c.attempts > 1 {
		var resp *http.Response
		var err error
		content, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(content))

		for i := 0; i < int(c.attempts); i++ {
			// canceled or deadline exceeded
			if ctx.Err() != nil {
				err = ctx.Err()
				break
			}
			resp, err = node.client.Do(req)
			if err == nil {
				break
			}
		}
		return resp, err
	}
	return node.client.Do(req)
}

// NewFactory new a client factory.
func NewFactory(logger log.Logger, r registry.Discovery) func(endpoint *config.Endpoint) (Client, error) {
	log := log.NewHelper(logger)
	return func(endpoint *config.Endpoint) (Client, error) {
		c := &clientImpl{
			selector: wrr.New(),
			attempts: 1,
		}
		timeout := endpoint.Timeout.AsDuration()
		if endpoint.Retry != nil {
			if endpoint.Retry.PerTryTimeout != nil && endpoint.Retry.PerTryTimeout.AsDuration() > 0 && endpoint.Retry.PerTryTimeout.AsDuration() < timeout {
				timeout = endpoint.Retry.PerTryTimeout.AsDuration()
			}
			if endpoint.Retry.Attempts > 1 {
				c.attempts = endpoint.Retry.Attempts
			}
			c.allowTriedNodes = endpoint.Retry.AllowTriedNodes
			c.conditions = endpoint.Retry.Conditions
		}

		nodes := []selector.Node{}
		atomicNodes := make(map[string]*node, 0)
		for _, backend := range endpoint.Backends {
			target, err := parseTarget(backend.Target)
			if err != nil {
				return nil, err
			}
			switch target.Scheme {
			case "direct":
				node := newNode(backend.Target, endpoint.Protocol, backend.Weight, timeout)
				nodes = append(nodes, node)
				atomicNodes[backend.Target] = node
			case "discovery":
				w, err := r.Watch(context.Background(), target.Endpoint)
				if err != nil {
					return nil, err
				}
				go func() {
					// TODO: goroutine leak
					// only one backend configuration allowed when using service discovery
					for {
						services, err := w.Next()
						if err != nil && errors.Is(err, context.Canceled) {
							return
						}
						if len(services) == 0 {
							continue
						}
						var nodes []selector.Node
						atomicNodes := make(map[string]*node, 0)
						for _, ser := range services {
							scheme := strings.ToLower(endpoint.Protocol.String())
							addr, err := parseEndpoint(ser.Endpoints, scheme, false)
							if err != nil {
								log.Errorf("failed to parse endpoint: %v", err)
								continue
							}
							node := newNode(addr, endpoint.Protocol, backend.Weight, timeout)
							nodes = append(nodes, node)
							atomicNodes[addr] = node
						}
						c.selector.Apply(nodes)
						c.nodes.Store(atomicNodes)
					}
				}()
			default:
				return nil, fmt.Errorf("unknown scheme: %s", target.Scheme)
			}
		}
		c.selector.Apply(nodes)
		c.nodes.Store(atomicNodes)
		return c, nil
	}
}
