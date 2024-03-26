package client

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/selector"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/http2"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

var _ selector.Node = &node{}
var _globalClient = defaultClient()
var _globalH2CClient = defaultH2CClient()
var _globalHTTPSClient = createHTTPSClient(nil)
var _dialTimeout = 200 * time.Millisecond
var followRedirect = false

func init() {
	var err error
	if v := os.Getenv("PROXY_DIAL_TIMEOUT"); v != "" {
		if _dialTimeout, err = time.ParseDuration(v); err != nil {
			panic(err)
		}
	}
	if val := os.Getenv("PROXY_FOLLOW_REDIRECT"); val != "" {
		followRedirect = true
	}
	prometheus.MustRegister(_metricClientRedirect)
}

var _metricClientRedirect = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "go",
	Subsystem: "gateway",
	Name:      "client_redirect_total",
	Help:      "The total number of client redirect",
}, []string{"protocol", "method", "path", "service", "basePath"})

func defaultCheckRedirect(req *http.Request, via []*http.Request) error {
	labels, ok := middleware.MetricsLabelsFromContext(req.Context())
	if ok {
		_metricClientRedirect.WithLabelValues(labels.Protocol(), labels.Method(), labels.Path(), labels.Service(), labels.BasePath()).Inc()
	}
	if followRedirect {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	return http.ErrUseLastResponse
}

func defaultClient() *http.Client {
	return &http.Client{
		CheckRedirect: defaultCheckRedirect,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   _dialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          10000,
			MaxIdleConnsPerHost:   1000,
			MaxConnsPerHost:       1000,
			DisableCompression:    true,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

func defaultH2CClient() *http.Client {
	return &http.Client{
		CheckRedirect: defaultCheckRedirect,
		Transport: &http2.Transport{
			// So http2.Transport doesn't complain the URL scheme isn't 'https'
			AllowHTTP:          true,
			DisableCompression: true,
			// Pretend we are dialing a TLS endpoint.
			// Note, we ignore the passed tls.Config
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.DialTimeout(network, addr, _dialTimeout)
			},
		},
	}
}

func createHTTPSClient(tlsConfig *tls.Config) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   _dialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          10000,
		MaxIdleConnsPerHost:   1000,
		MaxConnsPerHost:       1000,
		DisableCompression:    true,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	_ = http2.ConfigureTransport(tr)
	return &http.Client{
		CheckRedirect: defaultCheckRedirect,
		Transport:     tr,
	}
}

type HTTPSClientStore struct {
	clientConfigs map[string]*tls.Config
	clients       map[string]*http.Client
}

func NewHTTPSClientStore(clientConfigs map[string]*tls.Config) *HTTPSClientStore {
	return &HTTPSClientStore{
		clientConfigs: clientConfigs,
		clients:       make(map[string]*http.Client),
	}
}

func (s *HTTPSClientStore) GetClient(name string) *http.Client {
	if name == "" {
		return _globalClient
	}
	client, ok := s.clients[name]
	if ok {
		return client
	}
	tlsConfig, ok := s.clientConfigs[name]
	if !ok {
		LOG.Warnf("tls config not found for %s, using default instead", name)
		return _globalHTTPSClient
	}
	client = createHTTPSClient(tlsConfig)
	s.clients[name] = client
	return client
}

type NodeOptions struct {
	TLS           bool
	TLSConfigName string
}
type NewNodeOption func(*NodeOptions)

func WithTLS(in bool) NewNodeOption {
	return func(o *NodeOptions) {
		o.TLS = in
	}
}

func WithTLSConfigName(in string) NewNodeOption {
	return func(o *NodeOptions) {
		o.TLSConfigName = in
	}
}

func newNode(ctx *BuildContext, addr string, protocol config.Protocol, weight *int64, md map[string]string, version string, name string, opts ...NewNodeOption) *node {
	node := &node{
		protocol: protocol,
		address:  addr,
		weight:   weight,
		metadata: md,
		version:  version,
		name:     name,
	}
	node.client = _globalClient
	if protocol == config.Protocol_GRPC {
		node.client = _globalH2CClient
	}
	opt := &NodeOptions{}
	for _, o := range opts {
		o(opt)
	}
	if opt.TLS {
		node.tls = true
		node.client = _globalHTTPSClient
		if opt.TLSConfigName != "" {
			node.client = ctx.TLSClientStore.GetClient(opt.TLSConfigName)
		}
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
	protocol config.Protocol
	tls      bool
}

func (n *node) Scheme() string {
	return strings.ToLower(n.protocol.String())
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
