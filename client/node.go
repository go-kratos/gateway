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
var _gloalClient = defaultClient()
var _globalH2Client = defaultH2Client()

func defaultClient() *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.MaxIdleConns = 200
	tr.MaxConnsPerHost = 200
	tr.MaxIdleConnsPerHost = 200
	tr.DisableCompression = true
	return &http.Client{Transport: tr}
}

func defaultH2Client() *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			// So http2.Transport doesn't complain the URL scheme isn't 'https'
			AllowHTTP:          true,
			DisableCompression: true,
			// Pretend we are dialing a TLS endpoint.
			// Note, we ignore the passed tls.Config
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}
}

func globalClient() *http.Client {
	return _gloalClient
}

func globalH2Client() *http.Client {
	return _globalH2Client
}

type node struct {
	address  string
	name     string
	weight   *int64
	version  string
	metadata map[string]string

	client   *http.Client
	timeout  time.Duration
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

func newNode(addr string, protocol config.Protocol, weight *int64, timeout time.Duration, md map[string]string) *node {
	node := &node{
		protocol: protocol,
		address:  addr,
		weight:   weight,
		timeout:  timeout,
		metadata: md,
	}
	if protocol == config.Protocol_GRPC {
		node.client = globalH2Client()
	} else {
		node.client = globalClient()
	}
	return node
}
