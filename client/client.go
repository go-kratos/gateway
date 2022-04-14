package client

import (
	"context"
	"net/http"
	"strings"

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
	applier  *nodeApplier
	selector selector.Selector
}

func newClient(c *config.Endpoint, applier *nodeApplier, selector selector.Selector) *client {
	return &client{
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
	isGRPC := strings.HasPrefix(req.Header.Get("Content-Type"), "application/grpc")
	filter, _ := middleware.SelectorFiltersFromContext(ctx)
	filter = append(filter, func(ctx context.Context, nodes []selector.Node) []selector.Node {
		filtered := nodes[:0]
		for _, n := range nodes {
			if isGRPC && n.Scheme() == _schemeGRPC {
				filtered = append(filtered, n)
			} else {
				filtered = append(filtered, n)
			}
		}
		return filtered
	})
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
