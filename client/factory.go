package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/p2c"
)

// Factory is returns service client.
type Factory func(*config.Endpoint) (Client, error)

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
	return func(endpoint *config.Endpoint) (Client, error) {
		picker := o.pickerBuilder.Build()
		ctx, cancel := context.WithCancel(context.Background())
		applier := &nodeApplier{
			cancel:   cancel,
			endpoint: endpoint,
			registry: r,
			picker:   picker,
		}
		if err := applier.apply(ctx); err != nil {
			return nil, err
		}
		client := newClient(applier, picker)
		return client, nil
	}
}

type nodeApplier struct {
	canceled int64
	cancel   context.CancelFunc
	endpoint *config.Endpoint
	registry registry.Discovery
	picker   selector.Selector
}

func (na *nodeApplier) apply(ctx context.Context) error {
	var nodes []selector.Node
	for _, backend := range na.endpoint.Backends {
		target, err := parseTarget(backend.Target)
		if err != nil {
			return err
		}
		switch target.Scheme {
		case "direct":
			weighted := backend.Weight // weight is only valid for direct scheme
			node := newNode(backend.Target, na.endpoint.Protocol, weighted, map[string]string{}, "", "")
			nodes = append(nodes, node)
			na.picker.Apply(nodes)
		case "discovery":
			existed := AddWatch(ctx, na.registry, target.Endpoint, na)
			if existed {
				log.Infof("watch target %+v already existed", target)
			}
		default:
			return fmt.Errorf("unknown scheme: %s", target.Scheme)
		}
	}
	return nil
}

var _defaultWeight = int64(10)

func nodeWeight(n *registry.ServiceInstance) *int64 {
	w, ok := n.Metadata["weight"]
	if ok {
		val, _ := strconv.ParseInt(w, 10, 64)
		if val <= 0 {
			return &_defaultWeight
		}
		return &val
	}
	return &_defaultWeight
}

func (na *nodeApplier) Callback(services []*registry.ServiceInstance) error {
	if atomic.LoadInt64(&na.canceled) == 1 {
		return ErrCancelWatch
	}
	if len(services) == 0 {
		return nil
	}
	scheme := strings.ToLower(na.endpoint.Protocol.String())
	nodes := make([]selector.Node, 0, len(services))
	for _, ser := range services {
		addr, err := parseEndpoint(ser.Endpoints, scheme, false)
		if err != nil || addr == "" {
			log.Errorf("failed to parse endpoint: %v/%s: %v", ser.Endpoints, scheme, err)
			continue
		}
		node := newNode(addr, na.endpoint.Protocol, nodeWeight(ser), ser.Metadata, ser.Version, ser.Name)
		nodes = append(nodes, node)
	}
	na.picker.Apply(nodes)
	return nil
}

func (na *nodeApplier) Cancel() {
	log.Infof("Closing node applier for endpoint: %+v", na.endpoint)
	atomic.StoreInt64(&na.canceled, 1)
	na.cancel()
}

func (na *nodeApplier) Canceled() bool {
	return atomic.LoadInt64(&na.canceled) == 1
}
