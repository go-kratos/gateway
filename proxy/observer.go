package proxy

import (
	"net/http"
	"strconv"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_metricRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_code_total",
		Help:      "The total number of processed requests",
	}, []string{"protocol", "method", "path", "code", "service", "basePath"})
	_metricRequestsDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_duration_seconds",
		Help:      "Requests duration(sec).",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	}, []string{"protocol", "method", "path", "service", "basePath"})
	_metricSentBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_tx_bytes",
		Help:      "Total sent connection bytes",
	}, []string{"protocol", "method", "path", "service", "basePath"})
	_metricReceivedBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_rx_bytes",
		Help:      "Total received connection bytes",
	}, []string{"protocol", "method", "path", "service", "basePath"})
	_metricRetryState = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_retry_state",
		Help:      "Total request retries",
	}, []string{"protocol", "method", "path", "service", "basePath", "success"})
)

// Observable is the interface for observable proxy metrics.
type Observable interface {
	Observe(*config.Endpoint) Observer
}

// Observer is the interface for observing proxy metrics.
type Observer interface {
	HandleRetry(req *http.Request, state string)
	HandleRequest(req *http.Request, responseHeader http.Header, statusCode int)
	HandleSentBytes(req *http.Request, bytes int64)
	HandleReceivedBytes(req *http.Request, bytes int64)
	HandleLatency(req *http.Request, latency time.Duration)
}

// NewObservable creates a new Observable instance and registers the metrics.
func NewObservable() Observable {
	prometheus.MustRegister(_metricRequestsTotal)
	prometheus.MustRegister(_metricRequestsDuration)
	prometheus.MustRegister(_metricRetryState)
	prometheus.MustRegister(_metricSentBytes)
	prometheus.MustRegister(_metricReceivedBytes)
	return &observable{}
}

// NewObserver creates a new Observer instance and registers the metrics.
func NewObserver(endpoint *config.Endpoint) Observer {
	return &observer{}
}

type observable struct{}

func (o *observable) Observe(endpoint *config.Endpoint) Observer {
	return NewObserver(endpoint)
}

type observer struct {
	labels middleware.MetricsLabels
}

func (o *observer) HandleRequest(req *http.Request, responseHeader http.Header, statusCode int) {
	_metricRequestsTotal.WithLabelValues(o.labels.Protocol(), req.Method, o.labels.Path(), strconv.Itoa(statusCode), o.labels.Service(), o.labels.BasePath()).Inc()
}

func (o *observer) HandleRetry(req *http.Request, state string) {
	_metricRetryState.WithLabelValues(o.labels.Protocol(), req.Method, o.labels.Path(), o.labels.Service(), o.labels.BasePath(), state).Inc()
}

func (o *observer) HandleLatency(req *http.Request, latency time.Duration) {
	_metricRequestsDuration.WithLabelValues(o.labels.Protocol(), req.Method, o.labels.Path(), o.labels.Service(), o.labels.BasePath()).Observe(latency.Seconds())
}

func (o *observer) HandleSentBytes(req *http.Request, bytes int64) {
	_metricSentBytes.WithLabelValues(o.labels.Protocol(), req.Method, o.labels.Path(), o.labels.Service(), o.labels.BasePath()).Add(float64(bytes))
}

func (o *observer) HandleReceivedBytes(req *http.Request, bytes int64) {
	_metricReceivedBytes.WithLabelValues(o.labels.Protocol(), req.Method, o.labels.Path(), o.labels.Service(), o.labels.BasePath()).Add(float64(bytes))
}
