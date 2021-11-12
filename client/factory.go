package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/wrr"
)

// Factory is returns service client.
type Factory func(*config.Endpoint) (Client, error)

// NewFactory new a client factory.
func NewFactory(logger log.Logger, r registry.Discovery) Factory {
	log := log.NewHelper(logger)
	return func(endpoint *config.Endpoint) (Client, error) {
		wrr := wrr.New()
		applier := &nodeApplier{
			endpoint:  endpoint,
			logHelper: log,
			registry:  r,
		}
		applier.apply(context.Background(), wrr)

		client := &client{
			selector: wrr,
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
	endpoint  *config.Endpoint
	logHelper *log.Helper
	registry  registry.Discovery
}

func (na *nodeApplier) apply(ctx context.Context, dst selector.Selector) error {
	for _, backend := range na.endpoint.Backends {
		target, err := parseTarget(backend.Target)
		if err != nil {
			return err
		}
		switch target.Scheme {
		case "direct":
			nodes := []selector.Node{}
			atomicNodes := make(map[string]*node)
			node := newNode(backend.Target, na.endpoint.Protocol, backend.Weight, calcTimeout(na.endpoint))
			nodes = append(nodes, node)
			atomicNodes[backend.Target] = node
			dst.Apply(nodes)
		case "discovery":
			w, err := na.registry.Watch(ctx, target.Endpoint)
			if err != nil {
				return err
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
					atomicNodes := make(map[string]*node)
					for _, ser := range services {
						scheme := strings.ToLower(na.endpoint.Protocol.String())
						addr, err := parseEndpoint(ser.Endpoints, scheme, false)
						if err != nil {
							na.logHelper.Errorf("failed to parse endpoint: %v", err)
							continue
						}
						node := newNode(addr, na.endpoint.Protocol, backend.Weight, calcTimeout(na.endpoint))
						nodes = append(nodes, node)
						atomicNodes[addr] = node
					}
					dst.Apply(nodes)
				}
			}()
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
			cond.prepare()
			conditions = append(conditions, cond)
		case *config.RetryCondition_ByStatusCode:
			cond := &byStatusCode{
				RetryCondition_ByStatusCode: v,
			}
			cond.prepare()
			conditions = append(conditions, cond)
		default:
			return nil, fmt.Errorf("unknown condition type: %T", v)
		}
	}
	return conditions, nil
}
