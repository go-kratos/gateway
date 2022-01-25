package prometheus

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/prometheus"
	"github.com/go-kratos/gateway/middleware"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	once sync.Once

	seconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "duration_seconds",
		Help:      "Requests duration(sec).",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1},
	}, []string{"method", "path"})

	requests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_total",
		Help:      "The total number of processed requests",
	}, []string{"method", "path", "code"})
)

func init() {
	prometheus.MustRegister(seconds)
	prometheus.MustRegister(requests)
	middleware.Register("prometheus", Middleware)
}

// Middleware is a prometheus metrics.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Prometheus{
		Path: "/metrics",
	}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}
	once.Do(func() {
		http.Handle(options.Path, promhttp.Handler())
	})
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (reply *http.Response, err error) {
			startTime := time.Now()
			method := req.Method
			path := req.URL.Path
			code := http.StatusBadGateway
			reply, err = handler(ctx, req)
			if err == nil {
				code = reply.StatusCode
			}
			seconds.WithLabelValues(method, path).Observe(time.Since(startTime).Seconds())
			requests.WithLabelValues(method, path, strconv.Itoa(code)).Inc()
			return reply, err
		}
	}, nil
}
