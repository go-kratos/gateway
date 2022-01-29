package client

import (
	"crypto/tls"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/selector"
	"golang.org/x/net/http2"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
)

const _globalClientPool = 10

var (
	_                selector.Node = &node{}
	_globalClients   []*http.Client
	_globalH2Clients []*http.Client
)

func init() {
	for i := 0; i < _globalClientPool; i++ {
		_globalClients = append(_globalClients, defaultClient())
	}
	for i := 0; i < _globalClientPool; i++ {
		_globalH2Clients = append(_globalH2Clients, defaultH2Client())
	}
}

func defaultClient() *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.MaxIdleConns = 1000
	tr.MaxConnsPerHost = 100
	tr.MaxIdleConnsPerHost = 100
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
	return _globalClients[rand.Intn(_globalClientPool)]
}

func globalH2Client() *http.Client {
	return _globalH2Clients[rand.Intn(_globalClientPool)]
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
