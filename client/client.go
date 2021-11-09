package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"google.golang.org/grpc/codes"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/wrr"
)

// Client is a proxy client.
type Client interface {
	Invoke(ctx context.Context, req *http.Request) (*http.Response, error)
}

type clientFactoryContext struct {
	endpoint  *config.Endpoint
	logHelper *log.Helper
	registry  registry.Discovery
}

func (cfCtx *clientFactoryContext) applyNodes(dst selector.Selector) error {
	nodes := []selector.Node{}
	atomicNodes := make(map[string]*node)
	for _, backend := range cfCtx.endpoint.Backends {
		target, err := parseTarget(backend.Target)
		if err != nil {
			return err
		}
		switch target.Scheme {
		case "direct":
			node := newNode(backend.Target, cfCtx.endpoint.Protocol, backend.Weight, calcTimeout(cfCtx.endpoint))
			nodes = append(nodes, node)
			atomicNodes[backend.Target] = node
			dst.Apply(nodes)
		case "discovery":
			w, err := cfCtx.registry.Watch(context.Background(), target.Endpoint)
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
						scheme := strings.ToLower(cfCtx.endpoint.Protocol.String())
						addr, err := parseEndpoint(ser.Endpoints, scheme, false)
						if err != nil {
							cfCtx.logHelper.Errorf("failed to parse endpoint: %v", err)
							continue
						}
						node := newNode(addr, cfCtx.endpoint.Protocol, backend.Weight, calcTimeout(cfCtx.endpoint))
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

func parseRetryConditon(endpoint *config.Endpoint) ([][]uint32, error) {
	if endpoint.Retry == nil {
		return [][]uint32{}, nil
	}
	conditions := make([][]uint32, 0, len(endpoint.Retry.Conditions))
	for _, condition := range endpoint.Retry.Conditions {
		var statusCode []uint32
		switch endpoint.Protocol {
		case config.Protocol_GRPC:
			var code codes.Code
			if err := code.UnmarshalJSON([]byte(strings.ToUpper(condition))); err != nil {
				return nil, err
			}
			statusCode = append(statusCode, uint32(code))
		case config.Protocol_HTTP:
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
		conditions = append(conditions, statusCode)
	}
	return conditions, nil
}

// NewFactory new a client factory.
func NewFactory(logger log.Logger, r registry.Discovery) func(endpoint *config.Endpoint) (Client, error) {
	log := log.NewHelper(logger)
	return func(endpoint *config.Endpoint) (Client, error) {
		wrr := wrr.New()
		factoryCtx := &clientFactoryContext{
			endpoint:  endpoint,
			logHelper: log,
			registry:  r,
		}
		factoryCtx.applyNodes(wrr)

		client := &retryClient{
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
