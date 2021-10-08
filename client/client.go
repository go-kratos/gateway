package client

import (
	"context"
	"fmt"
	"log"
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"

	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/wrr"
)

// Client is a proxy client.
type Client interface {
	Invoke(ctx context.Context, req *http.Request, opts ...CallOption) (*http.Response, error)
}

type clientImpl struct {
	client   *http.Client
	selector selector.Selector
}

func (c *clientImpl) Invoke(ctx context.Context, req *http.Request, opts ...CallOption) (*http.Response, error) {
	callInfo := defaultCallInfo()
	for _, o := range opts {
		if err := o.before(&callInfo); err != nil {
			return nil, err
		}
	}

	selected, done, err := c.selector.Select(ctx, selector.WithFilter(callInfo.filters...))
	if err != nil {
		return nil, err
	}
	defer done(ctx, selector.DoneInfo{Err: err})
	scheme := req.URL.Scheme
	if scheme == "" {
		scheme = "http"
	}
	req, err = http.NewRequest(req.Method, fmt.Sprintf("%s://%s%s", scheme, selected.Address(), req.URL.RawPath), req.Body)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	log.Printf("invoke [%s] %s\n", req.Method, req.URL.Path)
	return resp, nil
}

// NewFactory new a client factory.
func NewFactory() func(backends []*config.Backend) (Client, error) {
	return func(backends []*config.Backend) (Client, error) {
		s := wrr.New()
		var nodes []selector.Node
		for _, backend := range backends {
			nodes = append(nodes, &node{address: backend.Target})
		}
		s.Apply(nodes)

		// TODO new a proxy client
		return &clientImpl{
			client:   &http.Client{},
			selector: s,
		}, nil
	}
}
