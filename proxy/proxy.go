package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/router"
	"github.com/go-kratos/gateway/router/mux"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/transport/http/status"
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
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1},
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

func init() {
	prometheus.MustRegister(_metricRequestsTotal)
	prometheus.MustRegister(_metricRequestsDuration)
	prometheus.MustRegister(_metricRetryState)
	prometheus.MustRegister(_metricSentBytes)
	prometheus.MustRegister(_metricReceivedBytes)
}

func setXFFHeader(req *http.Request) {
	// see https://github.com/golang/go/blob/master/src/net/http/httputil/reverseproxy.go
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		prior, ok := req.Header["X-Forwarded-For"]
		omit := ok && prior == nil // Issue 38079: nil now means don't populate the header
		if len(prior) > 0 {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		if !omit {
			req.Header.Set("X-Forwarded-For", clientIP)
		}
	}
}

func writeError(w http.ResponseWriter, r *http.Request, err error, labels middleware.MetricsLabels) {
	var statusCode int
	switch {
	case errors.Is(err, context.Canceled):
		statusCode = 499
	case errors.Is(err, context.DeadlineExceeded):
		statusCode = 504
	default:
		statusCode = 502
	}
	requestsTotalIncr(labels, statusCode)
	if labels.Protocol() == config.Protocol_GRPC.String() {
		// see https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
		code := strconv.Itoa(int(status.ToGRPCCode(statusCode)))
		w.Header().Set("Content-Type", "application/grpc")
		w.Header().Set("Grpc-Status", code)
		w.Header().Set("Grpc-Message", err.Error())
		statusCode = 200
	}
	w.WriteHeader(statusCode)
}

// notFoundHandler replies to the request with an HTTP 404 not found error.
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	code := http.StatusNotFound
	message := "404 page not found"
	http.Error(w, message, code)
	log.Context(r.Context()).Errorw(
		"source", "accesslog",
		"host", r.Host,
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"user_agent", r.Header.Get("User-Agent"),
		"code", code,
		"error", message,
	)
	_metricRequestsTotal.WithLabelValues("HTTP", r.Method, "/404", strconv.Itoa(code), "", "").Inc()
}

func methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	code := http.StatusMethodNotAllowed
	message := http.StatusText(code)
	http.Error(w, message, code)
	log.Context(r.Context()).Errorw(
		"source", "accesslog",
		"host", r.Host,
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"user_agent", r.Header.Get("User-Agent"),
		"code", code,
		"error", message,
	)
	_metricRequestsTotal.WithLabelValues("HTTP", r.Method, "/405", strconv.Itoa(code), "", "").Inc()
}

type interceptors struct {
	prepareAttemptTimeoutContext func(ctx context.Context, req *http.Request, timeout time.Duration) (context.Context, context.CancelFunc)
}

func (i *interceptors) SetPrepareAttemptTimeoutContext(f func(ctx context.Context, req *http.Request, timeout time.Duration) (context.Context, context.CancelFunc)) {
	if f != nil {
		i.prepareAttemptTimeoutContext = f
	}
}

// Proxy is a gateway proxy.
type Proxy struct {
	router            atomic.Value
	clientFactory     client.Factory
	Interceptors      interceptors
	middlewareFactory middleware.FactoryV2
}

// New is new a gateway proxy.
func New(clientFactory client.Factory, middlewareFactory middleware.FactoryV2) (*Proxy, error) {
	p := &Proxy{
		clientFactory:     clientFactory,
		middlewareFactory: middlewareFactory,
		Interceptors: interceptors{
			prepareAttemptTimeoutContext: defaultAttemptTimeoutContext,
		},
	}
	p.router.Store(mux.NewRouter(http.HandlerFunc(notFoundHandler), http.HandlerFunc(methodNotAllowedHandler)))
	return p, nil
}

func (p *Proxy) buildMiddleware(ms []*config.Middleware, next http.RoundTripper) (http.RoundTripper, error) {
	for i := len(ms) - 1; i >= 0; i-- {
		m, err := p.middlewareFactory(ms[i])
		if err != nil {
			if errors.Is(err, middleware.ErrNotFound) {
				log.Errorf("Skip does not exist middleware: %s", ms[i].Name)
				continue
			}
			return nil, err
		}
		next = m.Process(next)
	}
	return next, nil
}

func splitRetryMetricsHandler(e *config.Endpoint) (func(int), func(int, error)) {
	labels := middleware.NewMetricsLabels(e)
	success := func(i int) {
		if i <= 0 {
			return
		}
		retryStateIncr(labels, true)
	}
	failed := func(i int, err error) {
		if i <= 0 {
			return
		}
		if errors.Is(err, context.Canceled) {
			return
		}
		retryStateIncr(labels, false)
	}
	return success, failed
}

func (p *Proxy) buildEndpoint(e *config.Endpoint, ms []*config.Middleware) (http.Handler, io.Closer, error) {
	client, err := p.clientFactory(e)
	if err != nil {
		return nil, nil, err
	}
	tripper := http.RoundTripper(client)
	closer := io.Closer(client)
	tripper, err = p.buildMiddleware(e.Middlewares, tripper)
	if err != nil {
		return nil, nil, err
	}
	tripper, err = p.buildMiddleware(ms, tripper)
	if err != nil {
		return nil, nil, err
	}
	retryStrategy, err := prepareRetryStrategy(e)
	if err != nil {
		return nil, nil, err
	}
	labels := middleware.NewMetricsLabels(e)
	markSuccess, markFailed := splitRetryMetricsHandler(e)
	return http.Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		startTime := time.Now()
		setXFFHeader(req)

		reqOpts := middleware.NewRequestOptions(e)
		ctx := middleware.NewRequestContext(req.Context(), reqOpts)
		ctx, cancel := context.WithTimeout(ctx, retryStrategy.timeout)
		defer cancel()
		defer func() {
			requestsDurationObserve(labels, time.Since(startTime).Seconds())
		}()

		body, err := io.ReadAll(req.Body)
		if err != nil {
			writeError(w, req, err, labels)
			return
		}
		receivedBytesAdd(labels, int64(len(body)))
		req.GetBody = func() (io.ReadCloser, error) {
			reader := bytes.NewReader(body)
			return ioutil.NopCloser(reader), nil
		}

		var resp *http.Response
		for i := 0; i < retryStrategy.attempts; i++ {
			if (i + 1) >= retryStrategy.attempts {
				reqOpts.LastAttempt = true
			}
			// canceled or deadline exceeded
			if err = ctx.Err(); err != nil {
				markFailed(i, err)
				break
			}
			tryCtx, cancel := p.Interceptors.prepareAttemptTimeoutContext(ctx, req, retryStrategy.perTryTimeout)
			defer cancel()
			reader := bytes.NewReader(body)
			req.Body = ioutil.NopCloser(reader)
			resp, err = tripper.RoundTrip(req.Clone(tryCtx))
			if err != nil {
				markFailed(i, err)
				log.Errorf("Attempt at [%d/%d], failed to handle request: %s: %+v", i+1, retryStrategy.attempts, req.URL.String(), err)
				continue
			}
			if !judgeRetryRequired(retryStrategy.conditions, resp) {
				reqOpts.LastAttempt = true
				markSuccess(i)
				break
			}
			markFailed(i, errors.New("assertion failed"))
			// continue the retry loop
		}
		if err != nil {
			writeError(w, req, err, labels)
			return
		}

		headers := w.Header()
		for k, v := range resp.Header {
			headers[k] = v
		}
		w.WriteHeader(resp.StatusCode)

		doCopyBody := func() bool {
			if resp.Body == nil {
				return true
			}
			defer resp.Body.Close()
			sent, err := io.Copy(w, resp.Body)
			if err != nil {
				reqOpts.DoneFunc(ctx, selector.DoneInfo{Err: err})
				sentBytesAdd(labels, sent)
				log.Errorf("Failed to copy backend response body to client: [%s] %s %s %d %+v\n", e.Protocol, e.Method, e.Path, sent, err)
				return false
			}
			sentBytesAdd(labels, sent)
			reqOpts.DoneFunc(ctx, selector.DoneInfo{ReplyMD: resp.Trailer})
			// see https://pkg.go.dev/net/http#example-ResponseWriter-Trailers
			for k, v := range resp.Trailer {
				headers[http.TrailerPrefix+k] = v
			}
			return true
		}
		doCopyBody()
		requestsTotalIncr(labels, resp.StatusCode)
	})), closer, nil
}

func receivedBytesAdd(labels middleware.MetricsLabels, received int64) {
	_metricReceivedBytes.WithLabelValues(labels.Protocol(), labels.Method(), labels.Path(), labels.Service(), labels.BasePath()).Add(float64(received))
}

func sentBytesAdd(labels middleware.MetricsLabels, sent int64) {
	_metricSentBytes.WithLabelValues(labels.Protocol(), labels.Method(), labels.Path(), labels.Service(), labels.BasePath()).Add(float64(sent))
}

func requestsTotalIncr(labels middleware.MetricsLabels, statusCode int) {
	_metricRequestsTotal.WithLabelValues(labels.Protocol(), labels.Method(), labels.Path(), strconv.Itoa(statusCode), labels.Service(), labels.BasePath()).Inc()
}

func requestsDurationObserve(labels middleware.MetricsLabels, seconds float64) {
	_metricRequestsDuration.WithLabelValues(labels.Protocol(), labels.Method(), labels.Path(), labels.Service(), labels.BasePath()).Observe(seconds)
}

func retryStateIncr(labels middleware.MetricsLabels, success bool) {
	if success {
		_metricRetryState.WithLabelValues(labels.Protocol(), labels.Method(), labels.Path(), labels.Service(), labels.BasePath(), "true").Inc()
		return
	}
	_metricRetryState.WithLabelValues(labels.Protocol(), labels.Method(), labels.Path(), labels.Service(), labels.BasePath(), "false").Inc()
}

// Update updates service endpoint.
func (p *Proxy) Update(c *config.Gateway) error {
	router := mux.NewRouter(http.HandlerFunc(notFoundHandler), http.HandlerFunc(methodNotAllowedHandler))
	for _, e := range c.Endpoints {
		handler, closer, err := p.buildEndpoint(e, c.Middlewares)
		if err != nil {
			return err
		}
		if err = router.Handle(e.Path, e.Method, e.Host, handler, closer); err != nil {
			return err
		}
		log.Infof("build endpoint: [%s] %s %s", e.Protocol, e.Method, e.Path)
	}
	old := p.router.Swap(router)
	tryCloseRouter(old)
	return nil
}

func tryCloseRouter(in interface{}) {
	if in == nil {
		return
	}
	r, ok := in.(router.Router)
	if !ok {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		r.SyncClose(ctx)
	}()
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			w.WriteHeader(http.StatusBadGateway)
			buf := make([]byte, 64<<10) //nolint:gomnd
			n := runtime.Stack(buf, false)
			log.Errorf("panic recovered: %s", buf[:n])
			fmt.Fprintf(os.Stderr, "panic recovered: %s\n", buf[:n])
		}
	}()
	p.router.Load().(router.Router).ServeHTTP(w, req)
}

// DebugHandler implemented debug handler.
func (p *Proxy) DebugHandler() http.Handler {
	debugMux := http.NewServeMux()
	debugMux.HandleFunc("/debug/proxy/router/inspect", func(rw http.ResponseWriter, r *http.Request) {
		router, ok := p.router.Load().(router.Router)
		if !ok {
			return
		}
		inspect := mux.InspectMuxRouter(router)
		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(inspect)
	})
	return debugMux
}
