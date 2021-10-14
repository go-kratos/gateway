package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/selector"
	"golang.org/x/net/http2"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
)

var _ selector.Node = &node{}

type node struct {
	address  string
	name     string
	weight   *int64
	version  string
	metadata map[string]string

	client   *http.Client
	protocol config.Protocol
}

func (n *node) Address() string {
	return n.address
}

// ServiceName is service name
func (n *node) ServiceName() string {
	return n.name
}

// InitialWeight is the initial value of scheduling weight
// if not set return nil
func (n *node) InitialWeight() *int64 {
	return n.weight
}

// Version is service node version
func (n *node) Version() string {
	return n.version
}

// Metadata is the kv pair metadata associated with the service instance.
// version,namespace,region,protocol etc..
func (n *node) Metadata() map[string]string {
	return n.metadata
}

func newNode(addr string, protocol config.Protocol, weight *int64, timeout time.Duration) *node {
	client := &http.Client{
		Timeout:   timeout,
		Transport: http.DefaultTransport,
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
