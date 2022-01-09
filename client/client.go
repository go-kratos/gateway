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
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
	Close() error
}

type client struct {
	selector selector.Selector
	applier  *nodeApplier

	protocol   config.Protocol
	attempts   uint32
	conditions []retryCondition
}

func (c *client) Close() error {
	c.applier.Cancel()
	return nil
}

func (c *client) Do(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	req.URL.Scheme = "http"
	req.RequestURI = ""
	if c.attempts > 1 {
		return c.doRetry(ctx, req)
	}
	return c.do(ctx, req)
}

func (c *client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	opts, _ := middleware.FromRequestContext(ctx)
	selected, done, err := c.selector.Select(ctx, selector.WithFilter(opts.Filters...))
	if err != nil {
		return nil, err
	}
	defer done(ctx, selector.DoneInfo{Err: err})
	node := selected.(*node)
	req.URL.Host = selected.Address()
	ctx, cancel := context.WithTimeout(ctx, node.timeout)
	defer cancel()
	req = req.WithContext(ctx)
	return node.client.Do(req)
}

func duplicateRequestBody(req *http.Request) (*bytes.Reader, error) {
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

	selects := map[string]struct{}{}
	filter := func(ctx context.Context, nodes []selector.Node) []selector.Node {
		if len(selects) == 0 {
			return nodes
		}
		newNodes := nodes[:0]
		for _, node := range nodes {
			if _, ok := selects[node.Address()]; !ok {
				newNodes = append(newNodes, node)
			}
		}
		if len(newNodes) == 0 {
			return nodes
		}
		return newNodes
	}
	filters := opts.Filters
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

		selected, done, err := c.selector.Select(ctx, selector.WithFilter(filters...))
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
			// logging error
			continue
		}

		if !judgeRetryRequired(c.conditions, resp) {
			break
		}
		// continue the retry loop
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
