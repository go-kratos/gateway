package prometheus

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/prometheus"
	"github.com/go-kratos/gateway/middleware"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var once sync.Once

func init() {
	middleware.Register("prometheus", Middleware)
}

func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Prometheus{}
	if err := c.Options.UnmarshalTo(options); err != nil {
		return nil, err
	}
	seconds, err := newSeconds(options)
	if err != nil {
		return nil, err
	}
	requests, err := newRequests(options)
	once.Do(func() {
		http.Handle(options.Path, promhttp.Handler())
	})
	if err != nil {
		return nil, err
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (reply *http.Response, err error) {
			startTime := time.Now()
			reply, err = handler(ctx, req)
			var (
				code      string
				reason    string
				kind      string
				operation string
			)
			if req.ProtoMajor == 2 && strings.HasPrefix(req.Header.Get("Content-Type"), "application/json") {
				kind = "grpc"
			} else {
				kind = "http"
			}
			operation = req.URL.Path
			code = strconv.Itoa(reply.StatusCode)
			if err != nil {
				reason = err.Error()
			}
			seconds.With(kind, operation).Observe(time.Since(startTime).Seconds())
			requests.With(kind, operation, code, reason).Inc()
			return reply, err
		}
	}, nil
}

func newSeconds(opts *v1.Prometheus) (metrics.Observer, error) {
	s := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: opts.Namespace,
		Subsystem: "requests",
		Name:      "duration_sec",
		Help:      "requests duration(sec).",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1},
	}, []string{"kind", "operation"})
	return prom.NewHistogram(s), nil
}

func newRequests(opts *v1.Prometheus) (metrics.Counter, error) {
	r := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: opts.Namespace,
		Subsystem: "requests",
		Name:      "code_total",
		Help:      "The total number of processed requests",
	}, []string{"kind", "operation", "code", "reason"})
	return prom.NewCounter(r), nil
}
