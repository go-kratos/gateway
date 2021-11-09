package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/prometheus"
	"github.com/go-kratos/gateway/middleware"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	middleware.Register("prometheus", Middleware)
	http.Handle("/prometheus", promhttp.Handler())
}

func Middleware(ctx context.Context, c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Prometheus{}
	if err := c.Options.UnmarshalTo(options); err != nil {
		return nil, err
	}
	fmt.Println(options)
	seconds, err := newSeconds(options.Seconds)
	if err != nil {
		return nil, err
	}
	requests, err := newRequests(options.Requests)
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

func newSeconds(seconds *v1.Seconds) (metrics.Observer, error) {
	var buckets []float64
	for _, bucket := range seconds.Buckets {
		b, err := strconv.ParseFloat(bucket, 64)
		if err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	s := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   seconds.Namespace,
		Subsystem:   seconds.Subsystem,
		Name:        seconds.Name,
		Help:        seconds.Help,
		Buckets:     buckets,
		ConstLabels: seconds.ConstLabels,
	}, seconds.LabelNames)
	return prom.NewHistogram(s), nil
}

func newRequests(requests *v1.Requests) (metrics.Counter, error) {
	r := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   requests.Namespace,
		Subsystem:   requests.Subsystem,
		Name:        requests.Name,
		Help:        requests.Help,
		ConstLabels: requests.ConstLabels,
	}, requests.LabelNames)
	return prom.NewCounter(r), nil
}
