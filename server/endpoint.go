package server

import "net/url"

// NewEndpoint new an Endpoint URL.
func newEndpoint(scheme, host string, isSecure bool) *url.URL {
	var query string
	if isSecure {
		query = "isSecure=true"
	}
	return &url.URL{Scheme: scheme, Host: host, RawQuery: query}
}
