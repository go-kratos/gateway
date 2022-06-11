package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/p2c"
)

var (
	localIPAddress = loadLocalIPAddress()
)

// Factory is returns service client.
type Factory func(*config.Endpoint) (http.RoundTripper, error)

// NewFactory new a client factory.
func NewFactory(r registry.Discovery, proxyAddr string) Factory {
	return func(endpoint *config.Endpoint) (http.RoundTripper, error) {
		picker := p2c.New()
		ctx, cancel := context.WithCancel(context.Background())
		v, _ := net.ResolveTCPAddr("tcp", proxyAddr)
		applier := &nodeApplier{
			cancel:    cancel,
			endpoint:  endpoint,
			registry:  r,
			proxyPort: v.Port,
		}
		if err := applier.apply(ctx, picker); err != nil {
			return nil, err
		}
		return newClient(applier, picker), nil
	}
}

type nodeApplier struct {
	canceled  int64
	cancel    context.CancelFunc
	endpoint  *config.Endpoint
	registry  registry.Discovery
	proxyPort int
}

func (na *nodeApplier) apply(ctx context.Context, dst selector.Selector) error {
	var nodes []selector.Node
	for _, backend := range na.endpoint.Backends {
		target, err := parseTarget(backend.Target, na.proxyPort)
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
					addr, err := parseEndpoint(ser.Endpoints, scheme, false, na.proxyPort)
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

func loadLocalIPAddress() map[string]bool {
	ipAddress := make(map[string]bool)
	address, err := net.InterfaceAddrs()
	if err != nil {
		return ipAddress
	}
	for _, addr := range address {
		input, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if input.IP.To4() != nil {
			ipAddress[input.IP.To4().String()] = true
		}
		if input.IP.To16() != nil {
			ipAddress[input.IP.To16().String()] = true
		}
	}
	return ipAddress
}
