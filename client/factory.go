package client

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/p2c"
)

// Factory is returns service client.
type Factory func(*config.Endpoint) (Client, error)

// NewFactory new a client factory.
func NewFactory(r registry.Discovery) Factory {
	return func(endpoint *config.Endpoint) (Client, error) {
		picker := p2c.New()
		ctx, cancel := context.WithCancel(context.Background())
		applier := &nodeApplier{
			cancel:   cancel,
			endpoint: endpoint,
			registry: r,
		}
		if err := applier.apply(ctx, picker); err != nil {
			return nil, err
		}
		client := &client{
			selector: picker,
			applier:  applier,
			attempts: calcAttempts(endpoint),
			protocol: endpoint.Protocol,
		}
		retryCond, err := parseRetryConditon(endpoint)
		if err != nil {
			return nil, err
		}
		client.conditions = retryCond
		return client, nil
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
			node := newNode(backend.Target, na.endpoint.Protocol, weighted, calcTimeout(na.endpoint), map[string]string{})
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
						LOG.Errorf("failed to parse endpoint: %v", err)
						return nil
					}
					node := newNode(addr, na.endpoint.Protocol, weighted, calcTimeout(na.endpoint), ser.Metadata)
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

func calcTimeout(endpoint *config.Endpoint) time.Duration {
	timeout := endpoint.Timeout.AsDuration()
	if endpoint.Retry == nil {
		return timeout
	}
	if endpoint.Retry.PerTryTimeout != nil &&
		endpoint.Retry.PerTryTimeout.AsDuration() > 0 &&
		endpoint.Retry.PerTryTimeout.AsDuration() < timeout {
		return endpoint.Retry.PerTryTimeout.AsDuration()
	}
	return timeout
}

func calcAttempts(endpoint *config.Endpoint) uint32 {
	if endpoint.Retry == nil {
		return 1
	}
	if endpoint.Retry.Attempts == 0 {
		return 1
	}
	return endpoint.Retry.Attempts
}

func parseRetryConditon(endpoint *config.Endpoint) ([]retryCondition, error) {
	if endpoint.Retry == nil {
		return []retryCondition{}, nil
	}

	conditions := make([]retryCondition, 0, len(endpoint.Retry.Conditions))
	for _, rawCond := range endpoint.Retry.Conditions {
		switch v := rawCond.Condition.(type) {
		case *config.RetryCondition_ByHeader:
			cond := &byHeader{
				RetryCondition_ByHeader: v,
			}
			if err := cond.prepare(); err != nil {
				return nil, err
			}
			conditions = append(conditions, cond)
		case *config.RetryCondition_ByStatusCode:
			cond := &byStatusCode{
				RetryCondition_ByStatusCode: v,
			}
			if err := cond.prepare(); err != nil {
				return nil, err
			}
			conditions = append(conditions, cond)
		default:
			return nil, fmt.Errorf("unknown condition type: %T", v)
		}
	}
	return conditions, nil
}

func (na *nodeApplier) Cancel() {
	atomic.StoreInt64(&na.canceled, 1)
	na.cancel()
}
