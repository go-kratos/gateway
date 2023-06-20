package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/p2c"
)

// Factory is returns service client.
type Factory func(*config.Endpoint) (http.RoundTripper, ClientClose, error)
type ClientClose func() error

type Option func(*options)
type options struct {
	pickerBuilder selector.Builder
}

func WithPickerBuilder(in selector.Builder) Option {
	return func(o *options) {
		o.pickerBuilder = in
	}
}

// NewFactory new a client factory.
func NewFactory(r registry.Discovery, opts ...Option) Factory {
	o := &options{
		pickerBuilder: p2c.NewBuilder(),
	}
	for _, opt := range opts {
		opt(o)
	}
	return func(endpoint *config.Endpoint) (http.RoundTripper, ClientClose, error) {
		picker := o.pickerBuilder.Build()
		ctx, cancel := context.WithCancel(context.Background())
		applier := &nodeApplier{
			cancel:   cancel,
			endpoint: endpoint,
			registry: r,
		}
		if err := applier.apply(ctx, picker); err != nil {
			return nil, nil, err
		}
		client := newClient(applier, picker)
		return client, client.Close, nil
	}
}

type nodeApplier struct {
	canceled int64
	cancel   context.CancelFunc
	endpoint *config.Endpoint
	registry registry.Discovery
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
				var nodes []selector.Node
				for _, ser := range services {
					scheme := strings.ToLower(na.endpoint.Protocol.String())
					addr, err := parseEndpoint(ser.Endpoints, scheme, false)
					if err != nil || addr == "" {
						log.Errorf("failed to parse endpoint: %v", err)
						continue
					}
					node := newNode(addr, na.endpoint.Protocol, weighted, ser.Metadata)
					nodes = append(nodes, node)
				}
				dst.Apply(nodes)
				return nil
			})
			if existed {
				log.Infof("watch target %+v already existed", target)
			}
		default:
			return fmt.Errorf("unknown scheme: %s", target.Scheme)
		}
	}
	return nil
}

func (na *nodeApplier) Cancel() {
	log.Infof("Closing node applier for endpoint: %+v", na.endpoint)
	atomic.StoreInt64(&na.canceled, 1)
	na.cancel()
}
