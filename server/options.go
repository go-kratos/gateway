package server

import (
	"crypto/tls"
	"net/url"

	"github.com/go-kratos/kratos/v2/log"
)

// Option is an HTTP server option.
type Option func(*options)

type options struct {
	log      *log.Helper
	network  string
	endpoint *url.URL
	tlsConf  *tls.Config
}

// TLSConfig with server tls config.
func TLSConfig(tlsConf *tls.Config) Option {
	return func(o *options) {
		o.tlsConf = tlsConf
	}
}

// Network with server network.
func Network(network string) Option {
	return func(o *options) {
		o.network = network
	}
}

// Logger with server logger.
func Logger(logger log.Logger) Option {
	return func(o *options) {
		o.log = log.NewHelper(logger)
	}
}

// Endpoint with server endpoint.
func Endpoint(endpoint *url.URL) Option {
	return func(o *options) {
		o.endpoint = endpoint
	}
}
