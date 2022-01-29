package client

import (
	"context"
	"io"
	"net/http"
	"sync"

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

type retryClient struct {
	readers    *sync.Pool
	applier    *nodeApplier
	selector   selector.Selector
	protocol   config.Protocol
	attempts   uint32
	conditions []retryCondition
}

func newClient(c *config.Endpoint, applier *nodeApplier, selector selector.Selector) *retryClient {
	return &retryClient{
		protocol: c.Protocol,
		attempts: calcAttempts(c),
		applier:  applier,
		selector: selector,
		readers: &sync.Pool{
			New: func() interface{} {
				return &BodyReader{}
			},
		},
	}
}

func (c *retryClient) Close() error {
	c.applier.Cancel()
	return nil
}

func (c *retryClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.RequestURI = ""
	if c.attempts > 1 {
		return c.doRetry(ctx, req)
	}
	return c.do(ctx, req)
}

func (c *retryClient) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	opts, _ := middleware.FromRequestContext(ctx)
	n, done, err := c.selector.Select(ctx, selector.WithFilter(opts.Filters...))
	if err != nil {
		return nil, err
	}
	defer done(ctx, selector.DoneInfo{Err: err})
	node := n.(*node)
	req.URL.Host = n.Address()
	ctx, cancel := context.WithTimeout(ctx, node.timeout)
	defer cancel()
	req = req.WithContext(ctx)
	return node.client.Do(req)
}

func (c *retryClient) doRetry(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	var (
		opts, _  = middleware.FromRequestContext(ctx)
		selected = make(map[string]struct{}, 1)
	)
	opts.Filters = append(opts.Filters, func(ctx context.Context, nodes []selector.Node) []selector.Node {
		if len(selected) == 0 {
			return nodes
		}
		newNodes := nodes[:0]
		for _, node := range nodes {
			if _, ok := selected[node.Address()]; !ok {
				newNodes = append(newNodes, node)
			}
		}
		if len(newNodes) == 0 {
			return nodes
		}
		return newNodes
	})

	reader := c.readers.Get().(*BodyReader)
	if _, err := reader.ReadFrom(req.Body); err != nil {
		c.readers.Put(reader)
		return nil, err
	}
	req.Body = reader

	var (
		n    selector.Node
		done selector.DoneFunc
	)
	for i := 0; i < int(c.attempts); i++ {
		// canceled or deadline exceeded
		if err := ctx.Err(); err != nil {
			break
		}

		n, done, err = c.selector.Select(ctx, selector.WithFilter(opts.Filters...))
		if err != nil {
			break
		}
		addr := n.Address()
		reader.Seek(0, io.SeekStart)
		selected[addr] = struct{}{}
		req.URL.Host = addr
		resp, err = n.(*node).client.Do(req)
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
	c.readers.Put(reader)
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
