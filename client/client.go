package client

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// LOG .
	LOG = log.NewHelper(log.With(log.GetLogger(), "source", "client"))

	_metricRetries = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_retry_total",
		Help:      "The total number of retry requests",
	}, []string{"protocol", "method", "path"})
	_metricReceivedBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_rx_bytes",
		Help:      "Total received connection bytes",
	}, []string{"protocol", "method", "path"})
)

func init() {
	prometheus.MustRegister(_metricRetries)
	prometheus.MustRegister(_metricReceivedBytes)
}

// Client is a proxy client.
type Client interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
	Close() error
}

type retryClient struct {
	readers       *sync.Pool
	protocol      string
	applier       *nodeApplier
	selector      selector.Selector
	attempts      int
	timeout       time.Duration
	perTryTimeout time.Duration
	conditions    []retryCondition
}

func newClient(c *config.Endpoint, applier *nodeApplier, selector selector.Selector) *retryClient {
	return &retryClient{
		protocol:      c.Protocol.String(),
		timeout:       calcTimeout(c),
		attempts:      calcAttempts(c),
		perTryTimeout: calcPerTryTimeout(c),
		applier:       applier,
		selector:      selector,
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

func (c *retryClient) Do(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	opts, _ := middleware.FromRequestContext(ctx)
	selected := make(map[string]struct{}, 1)
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
	received, err := reader.ReadFrom(req.Body)
	if err != nil {
		c.readers.Put(reader)
		return nil, err
	}
	_metricReceivedBytes.WithLabelValues(c.protocol, req.Method, req.URL.Path).Add(float64(received))
	req.URL.Scheme = "http"
	req.RequestURI = ""
	req.Body = reader
	req.GetBody = func() (io.ReadCloser, error) {
		reader.Seek(0, io.SeekStart)
		return reader, nil
	}

	var (
		n    selector.Node
		done selector.DoneFunc
	)
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	for i := 0; i < c.attempts; i++ {
		// canceled or deadline exceeded
		if err := ctx.Err(); err != nil {
			break
		}
		rctx, cancel := context.WithTimeout(ctx, c.perTryTimeout)
		defer cancel()
		n, done, err = c.selector.Select(rctx, selector.WithFilter(opts.Filters...))
		if err != nil {
			break
		}
		addr := n.Address()
		opts.Backends = append(opts.Backends, addr)
		selected[addr] = struct{}{}
		req.URL.Host = addr
		req.GetBody() // seek reader to start
		resp, err = n.(*node).client.Do(req.WithContext(rctx))
		done(rctx, selector.DoneInfo{Err: err})
		if err != nil {
			// logging error
			// TODO: judge retry error
			_metricRetries.WithLabelValues(c.protocol, req.Method, req.URL.Path).Inc()
			continue
		}
		if !judgeRetryRequired(c.conditions, resp) {
			break
		}
		// continue the retry loop
		_metricRetries.WithLabelValues(c.protocol, req.Method, req.URL.Path).Inc()
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
