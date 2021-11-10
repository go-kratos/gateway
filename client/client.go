package client

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"strconv"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/selector"
)

// Client is a proxy client.
type Client interface {
	Invoke(ctx context.Context, req *http.Request) (*http.Response, error)
}

type client struct {
	selector selector.Selector

	protocol   config.Protocol
	attempts   uint32
	conditions [][]uint32
}

func (c *client) Invoke(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	// copy request to prevent body from being polluted
	req = req.WithContext(ctx)
	req.URL.Scheme = "http"
	req.RequestURI = ""
	if c.attempts > 1 {
		return c.doRetry(ctx, req)
	}
	return c.do(ctx, req)
}

func (c *client) do(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	opts, _ := middleware.FromRequestContext(ctx)
	selected, done, err := c.selector.Select(ctx, selector.WithNodeFilter(opts.Filters...))
	if err != nil {
		return nil, err
	}
	defer done(ctx, selector.DoneInfo{Err: err})
	node := selected.(*node)
	req.URL.Host = selected.Address()
	return node.client.Do(req)
}

func (c *client) doRetry(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	var content []byte
	var selects []string
	// TODO: get fixed bytes from pool if the content-length is specified
	content, err = ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	opts, _ := middleware.FromRequestContext(ctx)
	filters := opts.Filters
	filter := func(node selector.Node) bool {
		for _, s := range selects {
			if node.Address() == s {
				return false
			}
		}
		return true
	}

	filters = append(filters, filter)

	for i := 0; i < int(c.attempts); i++ {
		// canceled or deadline exceeded
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}

		selected, done, err := c.selector.Select(ctx, selector.WithNodeFilter(filters...))
		if err != nil {
			break
		}
		addr := selected.Address()
		selects = append(selects, addr)
		req.URL.Host = selected.Address()
		req.Body = ioutil.NopCloser(bytes.NewReader(content))
		resp, err = selected.(*node).client.Do(req)
		done(ctx, selector.DoneInfo{Err: err})
		if err != nil {
			continue
		}

		var statusCode uint32
		if c.protocol == config.Protocol_GRPC {
			if resp.StatusCode != 200 {
				continue
			}
			code, _ := strconv.ParseInt(resp.Header.Get("Grpc-Status"), 10, 64)
			statusCode = uint32(code)
		} else {
			statusCode = uint32(resp.StatusCode)
		}
		for _, condition := range c.conditions {
			if len(condition) == 1 {
				if condition[0] == statusCode {
					continue
				} else if statusCode >= condition[0] && statusCode <= condition[1] {
					continue
				}
			}
		}

		// err is nil and no status-conditions is hitted
		break
	}
	return
}
