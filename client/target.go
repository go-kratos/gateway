package client

import (
	"errors"
	"net"
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

func parseTarget(endpoint string, port int) (*Target, error) {
	if err := checkEndpoint(endpoint, port); err != nil {
		return nil, err
	}
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
func parseEndpoint(endpoints []string, scheme string, isSecure bool, port int) (string, error) {
	for _, e := range endpoints {
		if err := checkEndpoint(e, port); err != nil {
			return "", err
		}
		u, err := url.Parse(e)
		if err != nil {
			return "", err
		}
		if u.Scheme == scheme {
			if IsSecure(u) == isSecure {
				return u.Host, nil
			}
		}
	}
	return "", nil
}

// IsSecure parses isSecure for Endpoint URL.
func IsSecure(u *url.URL) bool {
	ok, err := strconv.ParseBool(u.Query().Get("isSecure"))
	if err != nil {
		return false
	}
	return ok
}

func checkEndpoint(endpoint string, servicePort int) error {
	if strings.HasPrefix(endpoint, "direct:///") {
		endpoint = strings.TrimPrefix(endpoint, "direct:///")
	}
	if strings.HasPrefix(endpoint, "discovery:///") {
		endpoint = strings.TrimPrefix(endpoint, "discovery:///")
	}
	if endpoint == "" {
		return errors.New("endpoint must not be empty")
	}
	v, err := net.ResolveTCPAddr("tcp", endpoint)
	if err != nil && endpoint != "localhost" {
		return err
	}
	if (endpoint == "localhost" && servicePort == 80) || (endpoint != "localhost" && localIPAddress[v.IP.String()] && servicePort == v.Port) {
		return errors.New("endpoint port must not be same as service port")
	}
	return nil
}
