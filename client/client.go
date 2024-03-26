package client

import (
	"io"
	"net/http"
	"time"

	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/selector"
)

type client struct {
	applier  *nodeApplier
	selector selector.Selector
}

type Client interface {
	http.RoundTripper
	io.Closer
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
	reqOpt, _ := middleware.FromRequestContext(ctx)
	filter, _ := middleware.SelectorFiltersFromContext(ctx)
	n, done, err := c.selector.Select(ctx, selector.WithNodeFilter(filter...))
	if err != nil {
		return nil, err
	}
	reqOpt.CurrentNode = n

	addr := n.Address()
	reqOpt.Backends = append(reqOpt.Backends, addr)
	backendNode := n.(*node)
	req.URL.Host = addr
	req.URL.Scheme = "http"
	if backendNode.tls {
		req.URL.Scheme = "https"
		if req.Host == "" {
			req.Host = addr
		}
	}
	req.RequestURI = ""
	startAt := time.Now()
	resp, err = backendNode.client.Do(req)
	reqOpt.UpstreamResponseTime = append(reqOpt.UpstreamResponseTime, time.Since(startAt).Seconds())
	if err != nil {
		done(ctx, selector.DoneInfo{Err: err})
		reqOpt.UpstreamStatusCode = append(reqOpt.UpstreamStatusCode, 0)
		return nil, err
	}
	reqOpt.UpstreamStatusCode = append(reqOpt.UpstreamStatusCode, resp.StatusCode)
	reqOpt.DoneFunc = done
	return resp, nil
}
