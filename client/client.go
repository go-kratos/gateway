package client

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"

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
	conditions []retryCondition
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

func duplicateRequestBody(req *http.Request) (*bytes.Reader, error) {
	// TODO: get fixed bytes from pool if the content-length is specified
	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(content)
	req.Body = ioutil.NopCloser(body)
	return body, nil
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

	body, err := duplicateRequestBody(req)
	if err != nil {
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
		body.Seek(0, io.SeekStart)
		selects[addr] = struct{}{}
		req.URL.Host = addr
		resp, err = selected.(*node).client.Do(req)
		done(ctx, selector.DoneInfo{Err: err})
		if err != nil {
			// logging
			continue
		}

		if judgeRetryRequired(c.conditions, resp) {
			// continue the retry loop
			continue
		}
	}
	return
}

func judgeRetryRequired(conditions []retryCondition, resp *http.Response) bool {
	for _, cond := range conditions {
		if cond.judge(resp) {
			return true
		}
	}
	return false
}
