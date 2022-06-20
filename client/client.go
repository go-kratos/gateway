package client

import (
	"net/http"

	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/selector"
)

type client struct {
	applier  *nodeApplier
	selector selector.Selector
}

func newClient(applier *nodeApplier, selector selector.Selector) *client {
	return &client{
		applier:  applier,
		selector: selector,
	}
}

func (c *client) Close() error {
	c.applier.Cancel()
	return nil
}

func (c *client) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	ctx := req.Context()
	filter, _ := middleware.SelectorFiltersFromContext(ctx)
	n, done, err := c.selector.Select(ctx, selector.WithFilter(filter...))
	if err != nil {
		return nil, err
	}
	addr := n.Address()
	middleware.WithRequestBackends(ctx, addr)
	req.URL.Host = addr
	req.URL.Scheme = "http"
	req.RequestURI = ""
	resp, err = n.(*node).client.Do(req)
	done(ctx, selector.DoneInfo{Err: err})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
