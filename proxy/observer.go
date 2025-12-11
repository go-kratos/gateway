package proxy

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_code_total",
		Help:      "The total number of processed requests",
	}, []string{"protocol", "method", "path", "code", "service", "basePath"})
	metricRequestsDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_duration_seconds",
		Help:      "Requests duration(sec).",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	}, []string{"protocol", "method", "path", "service", "basePath"})
	metricSentBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_tx_bytes",
		Help:      "Total sent connection bytes",
	}, []string{"protocol", "method", "path", "service", "basePath"})
	metricReceivedBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_rx_bytes",
		Help:      "Total received connection bytes",
	}, []string{"protocol", "method", "path", "service", "basePath"})
	metricRetryState = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_retry_state",
		Help:      "Total request retries",
	}, []string{"protocol", "method", "path", "service", "basePath", "success"})
	// ensure the metric is registered only once
	metricOnce sync.Once
)

// Observable is the interface for observable proxy metrics.
type Observable interface {
	Observe(*config.Endpoint) Observer
}

// Observer is the interface for observing proxy metrics.
type Observer interface {
	HandleRetry(req *http.Request, responseHeader http.Header, state string)
	HandleRequest(req *http.Request, responseHeader http.Header, statusCode int)
	HandleSentBytes(req *http.Request, bytes int64)
	HandleReceivedBytes(req *http.Request, bytes int64)
	HandleLatency(req *http.Request, latency time.Duration)
}

// NewObservable creates a new Observable instance and registers the metrics.
func NewObservable() Observable {
	metricOnce.Do(func() {
		prometheus.MustRegister(metricRequestsTotal)
		prometheus.MustRegister(metricRequestsDuration)
		prometheus.MustRegister(metricRetryState)
		prometheus.MustRegister(metricSentBytes)
		prometheus.MustRegister(metricReceivedBytes)
	})
	return &observable{}
}

// NewObserver creates a new Observer instance and registers the metrics.
func NewObserver(endpoint *config.Endpoint) Observer {
	return &observer{}
}

type observerKey struct{}

type observable struct{}

func (o *observable) Observe(endpoint *config.Endpoint) Observer {
	return NewObserver(endpoint)
}

type observer struct {
	labels middleware.MetricsLabels
}

func (o *observer) HandleRequest(req *http.Request, responseHeader http.Header, statusCode int) {
	metricRequestsTotal.WithLabelValues(o.labels.Protocol(), req.Method, o.labels.Path(), strconv.Itoa(statusCode), o.labels.Service(), o.labels.BasePath()).Inc()
}

func (o *observer) HandleRetry(req *http.Request, responseHeader http.Header, state string) {
	metricRetryState.WithLabelValues(o.labels.Protocol(), req.Method, o.labels.Path(), o.labels.Service(), o.labels.BasePath(), state).Inc()
}

func (o *observer) HandleLatency(req *http.Request, latency time.Duration) {
	metricRequestsDuration.WithLabelValues(o.labels.Protocol(), req.Method, o.labels.Path(), o.labels.Service(), o.labels.BasePath()).Observe(latency.Seconds())
}

func (o *observer) HandleSentBytes(req *http.Request, bytes int64) {
	metricSentBytes.WithLabelValues(o.labels.Protocol(), req.Method, o.labels.Path(), o.labels.Service(), o.labels.BasePath()).Add(float64(bytes))
}

func (o *observer) HandleReceivedBytes(req *http.Request, bytes int64) {
	metricReceivedBytes.WithLabelValues(o.labels.Protocol(), req.Method, o.labels.Path(), o.labels.Service(), o.labels.BasePath()).Add(float64(bytes))
}
