package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/endpoint"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/node/direct"
	"github.com/go-kratos/kratos/v2/selector/wrr"
	"google.golang.org/grpc/codes"
)

// Client is a proxy client.
type Client interface {
	Invoke(ctx context.Context, req *http.Request) (*http.Response, error)
}

type client struct {
	selector selector.Selector
}

func (c *client) Invoke(ctx context.Context, req *http.Request) (*http.Response, error) {
	opts, _ := endpoint.FromContext(ctx)
	selected, done, err := c.selector.Select(ctx, selector.WithFilter(opts.Filters...))
	if err != nil {
		return nil, err
	}
	defer done(ctx, selector.DoneInfo{Err: err})

	wn := selected.(*direct.Node)
	node := wn.Node.(*node)
	req = req.WithContext(ctx)
	req.URL.Scheme = "http"
	req.URL.Host = selected.Address()
	req.RequestURI = ""

	return node.client.Do(req)
}

// NewFactory new a client factory.
func NewFactory(logger log.Logger, r registry.Discovery) func(endpoint *config.Endpoint) (Client, error) {
	log := log.NewHelper(logger)
	return func(endpoint *config.Endpoint) (Client, error) {
		var c Client
		timeout := endpoint.Timeout.AsDuration()
		wrr := wrr.New()

		if endpoint.Retry != nil && endpoint.Retry.Attempts > 1 {
			if endpoint.Retry.PerTryTimeout != nil && endpoint.Retry.PerTryTimeout.AsDuration() > 0 && endpoint.Retry.PerTryTimeout.AsDuration() < timeout {
				timeout = endpoint.Retry.PerTryTimeout.AsDuration()
			}
			rc := &retryClient{
				selector: wrr,
				attempts: 1,
				protocol: endpoint.Protocol,
			}
			rc.attempts = endpoint.Retry.Attempts
			rc.allowTriedNodes = endpoint.Retry.AllowTriedNodes
			for _, condition := range endpoint.Retry.Conditions {
				var statusCode []uint32
				if endpoint.Protocol == config.Protocol_GRPC {
					var code codes.Code
					err := code.UnmarshalJSON([]byte(strings.ToUpper(condition)))
					if err != nil {
						return nil, err
					}
					statusCode = append(statusCode, uint32(code))
				} else {
					cs := strings.Split(condition, "-")
					if len(cs) == 0 || len(cs) > 2 {
						return nil, fmt.Errorf("invalid condition %s", condition)
					}
					for _, c := range cs {
						code, err := strconv.ParseUint(c, 10, 16)
						if err != nil {
							return nil, err
						}
						statusCode = append(statusCode, uint32(code))
					}
				}
				rc.conditions = append(rc.conditions, statusCode)
			}
			c = rc
		} else {
			if endpoint.Retry != nil && endpoint.Retry.PerTryTimeout != nil && endpoint.Retry.PerTryTimeout.AsDuration() > 0 && endpoint.Retry.PerTryTimeout.AsDuration() < timeout {
				timeout = endpoint.Retry.PerTryTimeout.AsDuration()
			}
			c = &client{
				selector: wrr,
			}
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
						wrr.Apply(nodes)
					}
				}()
			default:
				return nil, fmt.Errorf("unknown scheme: %s", target.Scheme)
			}
		}
		wrr.Apply(nodes)
		return c, nil
	}
}
