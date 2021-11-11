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

func (c *client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
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

func duplicateRequestBody(ctx context.Context, req *http.Request) error {
	// TODO: get fixed bytes from pool if the content-length is specified
	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	// copy request to prevent bdoy from being polluted
	req.Body = ioutil.NopCloser(bytes.NewReader(content))
	return nil
}

func (c *client) doRetry(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	opts, _ := middleware.FromRequestContext(ctx)
	filters := opts.Filters

	selects := map[string]struct{}{}
	filter := func(node selector.Node) bool {
		if _, ok := selects[node.Address()]; ok {
			return false
		}
		return true
	}
	filters = append(filters, filter)

	if err := duplicateRequestBody(ctx, req); err != nil {
		return nil, err
	}
	for i := 0; i < int(c.attempts); i++ {
		// canceled or deadline exceeded
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		selected, done, err := c.selector.Select(ctx, selector.WithNodeFilter(filters...))
		if err != nil {
			return nil, err
		}
		addr := selected.Address()
		selects[addr] = struct{}{}
		req.URL.Host = addr
		resp, err = selected.(*node).client.Do(req)
		done(ctx, selector.DoneInfo{Err: err})
		if err != nil {
			// logging
			continue
		}

		statusCode := parseStatusCode(resp, c.protocol)
		if judgeRetryRequired(c.conditions, statusCode) {
			// continue the retry loop
			continue
		}
	}
	return
}

func parseStatusCode(resp *http.Response, protocol config.Protocol) uint32 {
	if protocol == config.Protocol_GRPC {
		code, _ := strconv.ParseInt(resp.Header.Get("Grpc-Status"), 10, 64)
		return uint32(code)
	}
	return uint32(resp.StatusCode)
}

func judgeRetryRequired(conditions [][]uint32, statusCode uint32) bool {
	for _, condition := range conditions {
		if len(condition) == 1 {
			if condition[0] == statusCode {
				return true
			} else if statusCode >= condition[0] && statusCode <= condition[1] {
				return true
			}
		}
	}
	return false
}
