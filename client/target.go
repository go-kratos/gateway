package client

import (
	"net/url"
	"strconv"
	"strings"
)

// Target is resolver target
type Target struct {
	Scheme    string
	Authority string
	Endpoint  string
}

func parseTarget(endpoint string) (*Target, error) {
	if !strings.Contains(endpoint, "://") {
		endpoint = "direct:///" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	target := &Target{Scheme: u.Scheme, Authority: u.Host}
	if len(u.Path) > 1 {
		target.Endpoint = u.Path[1:]
	}
	return target, nil
}

// parseEndpoint parses an Endpoint URL.
func parseEndpoint(endpoints []string, isSecure bool) (urls []*url.URL, err error) {
	for _, e := range endpoints {
		u, err := url.Parse(e)
		if err != nil {
			return nil, err
		}
		if IsSecure(u) == isSecure {
			urls = append(urls, u)
		}
	}
	return nil, nil
}

// IsSecure parses isSecure for Endpoint URL.
func IsSecure(u *url.URL) bool {
	ok, err := strconv.ParseBool(u.Query().Get("isSecure"))
	if err != nil {
		return false
	}
	return ok
}
