package client

import (
	"context"
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/selector"
)

var (
	// LOG .
	LOG = log.NewHelper(log.With(log.GetLogger(), "source", "client"))
)

// Client is a proxy client.
type Client interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
	Close() error
}

type client struct {
	// readers  *sync.Pool
	protocol string
	applier  *nodeApplier
	selector selector.Selector
}

func newClient(c *config.Endpoint, applier *nodeApplier, selector selector.Selector) *client {
	return &client{
		protocol: c.Protocol.String(),
		applier:  applier,
		selector: selector,
	}
}

func (c *client) Close() error {
	c.applier.Cancel()
	return nil
}

func (c *client) Do(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	req.URL.Scheme = "http"
	req.RequestURI = ""
	filter, _ := middleware.SelectorFiltersFromContext(ctx)
	n, done, err := c.selector.Select(ctx, selector.WithFilter(filter...))
	if err != nil {
		return nil, err
	}
	addr := n.Address()
	middleware.WithRequestBackends(ctx, addr)
	req.URL.Host = addr
	resp, err = n.(*node).client.Do(req.WithContext(ctx))
	done(ctx, selector.DoneInfo{Err: err})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
