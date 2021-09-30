package backend

import (
	"fmt"
	"net/http"

	"github.com/go-kratos/kratos/v2/selector"
)

var _ selector.Node = &node{}

type node struct {
	address  string
	name     string
	weight   *int64
	version  string
	metadata map[string]string
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

type Client struct {
	client   *http.Client
	selector selector.Selector
	scheme   string
}

func (c *Client) Do(r *http.Request) (*http.Response, error) {
	selected, done, err := c.selector.Select(r.Context())
	if err != nil {
		return nil, err
	}
	defer done(r.Context(), selector.DoneInfo{Err: err})
	scheme := r.URL.Scheme
	if c.scheme != "" {
		scheme = c.scheme
	}
	req, err := http.NewRequest(r.Method, fmt.Sprintf("%s://%s%s", scheme, selected.Address(), r.URL.RawPath), r.Body)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, err
}
