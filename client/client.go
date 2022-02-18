package client

import (
	"context"
	"io"
	"net/http"
	"sync"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// LOG .
	LOG = log.NewHelper(log.With(log.GetLogger(), "source", "client"))

	_metricRetryTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_retry_total",
		Help:      "Total request retries",
	}, []string{"protocol", "method", "path"})
	_metricRetrySuccess = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_retry_success",
		Help:      "Total request retry successes",
	}, []string{"protocol", "method", "path"})
	_metricReceivedBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_rx_bytes",
		Help:      "Total received connection bytes",
	}, []string{"protocol", "method", "path"})
)

func init() {
	prometheus.MustRegister(_metricRetryTotal)
	prometheus.MustRegister(_metricRetrySuccess)
	prometheus.MustRegister(_metricReceivedBytes)
}

// Client is a proxy client.
type Client interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
	Close() error
}

type retryClient struct {
	readers  *sync.Pool
	protocol string
	applier  *nodeApplier
	selector selector.Selector
	// attempts int
	// timeout       time.Duration
	// perTryTimeout time.Duration
	// conditions []retryCondition
}

func newClient(c *config.Endpoint, applier *nodeApplier, selector selector.Selector) *retryClient {
	return &retryClient{
		protocol: c.Protocol.String(),
		// timeout:       calcTimeout(c),
		// attempts: calcAttempts(c),
		// perTryTimeout: calcPerTryTimeout(c),
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

func (c *retryClient) Do(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
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

	n, done, err := c.selector.Select(ctx)
	if err != nil {
		return nil, err
	}
	addr := n.Address()
	req.URL.Host = addr
	req.Host = addr
	req.GetBody() // seek reader to start
	resp, err = n.(*node).client.Do(req.WithContext(ctx))
	done(ctx, selector.DoneInfo{Err: err})
	if err != nil {
		return nil, err
	}
	c.readers.Put(reader)
	return resp, nil
}

func judgeRetryRequired(conditions []retryCondition, resp *http.Response) bool {
	for _, cond := range conditions {
		if cond.judge(resp) {
			return true
		}
	}
	return false
}
