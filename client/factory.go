package client

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/p2c"
)

type options struct {
	subsetFn subsetFn
}

type Option func(*options)

func WithSubsetFn(in subsetFn) Option {
	return func(o *options) {
		o.subsetFn = in
	}
}

// Factory is returns service client.
type Factory func(*config.Endpoint) (Client, error)

// NewFactory new a client factory.
func NewFactory(r registry.Discovery, opts ...Option) Factory {
	o := &options{
		subsetFn: makeSubsetFn(20),
	}
	for _, opt := range opts {
		opt(o)
	}
	return func(endpoint *config.Endpoint) (Client, error) {
		picker := p2c.New()
		ctx, cancel := context.WithCancel(context.Background())
		applier := &nodeApplier{
			cancel:   cancel,
			endpoint: endpoint,
			registry: r,
			subsetFn: o.subsetFn,
		}
		if err := applier.apply(ctx, picker); err != nil {
			return nil, err
		}
		return newClient(endpoint, applier, picker), nil
	}
}

type nodeApplier struct {
	canceled int64
	cancel   context.CancelFunc
	endpoint *config.Endpoint
	registry registry.Discovery
	subsetFn subsetFn
}

func (na *nodeApplier) apply(ctx context.Context, dst selector.Selector) error {
	var nodes []selector.Node
	for _, backend := range na.endpoint.Backends {
		target, err := parseTarget(backend.Target)
		if err != nil {
			return err
		}
		weighted := backend.Weight
		switch target.Scheme {
		case "direct":
			node := newNode(backend.Target, na.endpoint.Protocol, weighted, map[string]string{})
			nodes = append(nodes, node)
			dst.Apply(nodes)
		case "discovery":
			existed := AddWatch(ctx, na.registry, target.Endpoint, func(services []*registry.ServiceInstance) error {
				if atomic.LoadInt64(&na.canceled) == 1 {
					return ErrCancelWatch
				}
				if len(services) == 0 {
					return nil
				}
				if na.subsetFn != nil {
					services = na.subsetFn(services)
				}
				var nodes []selector.Node
				for _, ser := range services {
					scheme := strings.ToLower(na.endpoint.Protocol.String())
					addr, err := parseEndpoint(ser.Endpoints, scheme, false)
					if err != nil || addr == "" {
						LOG.Errorf("failed to parse endpoint: %v", err)
						return nil
					}
					node := newNode(addr, na.endpoint.Protocol, weighted, ser.Metadata)
					nodes = append(nodes, node)
				}
				dst.Apply(nodes)
				return nil
			})
			if existed {
				LOG.Infof("watch target %+v already existed", target)
			}
		default:
			return fmt.Errorf("unknown scheme: %s", target.Scheme)
		}
	}
	return nil
}

func (na *nodeApplier) Cancel() {
	atomic.StoreInt64(&na.canceled, 1)
	na.cancel()
}
