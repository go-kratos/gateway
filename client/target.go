package client

import (
	"math/rand"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dgryski/go-farm"
	"github.com/go-kratos/kratos/v2/registry"
)

// Target is resolver target
type Target struct {
	Scheme    string
	Authority string
	Endpoint  string
}

type subsetFn func(instances []*registry.ServiceInstance, size int) []*registry.ServiceInstance

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
func parseEndpoint(endpoints []string, scheme string, isSecure bool) (string, error) {
	for _, e := range endpoints {
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

func genClientID() string {
	hostname := os.Getenv("HOSTNAME")
	if hostname != "" {
		return hostname
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = strconv.Itoa(int(time.Now().UnixNano()))
	}
	return hostname
}

func defaultSubset(instances []*registry.ServiceInstance, size int) []*registry.ServiceInstance {
	backends := instances
	if size <= 0 {
		return backends
	}
	if len(backends) <= int(size) {
		return backends
	}
	clientID := genClientID()
	sort.Slice(backends, func(i, j int) bool {
		return backends[i].ID < backends[j].ID
	})
	count := len(backends) / size
	// hash得到ID
	id := farm.Fingerprint64([]byte(clientID))
	// 获得rand轮数
	round := int64(id / uint64(count))

	s := rand.NewSource(round)
	ra := rand.New(s)
	//  根据source洗牌
	ra.Shuffle(len(backends), func(i, j int) {
		backends[i], backends[j] = backends[j], backends[i]
	})
	start := (id % uint64(count)) * uint64(size)
	return backends[int(start) : int(start)+int(size)]
}
