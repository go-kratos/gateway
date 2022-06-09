package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/router"
	"github.com/go-kratos/gateway/router/mux"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http/status"
	gorillamux "github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

// LOG .
var LOG = log.NewHelper(log.With(log.GetLogger(), "source", "proxy"))

var (
	_metricRequestsTotol = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_code_total",
		Help:      "The total number of processed requests",
	}, []string{"protocol", "method", "path", "code"})
	_metricRequestsDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_duration_seconds",
		Help:      "Requests duration(sec).",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1},
	}, []string{"protocol", "method", "path"})
	_metricSentBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_tx_bytes",
		Help:      "Total sent connection bytes",
	}, []string{"protocol", "method", "path"})
	_metricReceivedBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_rx_bytes",
		Help:      "Total received connection bytes",
	}, []string{"protocol", "method", "path"})
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
)

func init() {
	prometheus.MustRegister(_metricRequestsTotol)
	prometheus.MustRegister(_metricRequestsDuration)
	prometheus.MustRegister(_metricRetryTotal)
	prometheus.MustRegister(_metricRetrySuccess)
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

func writeError(w http.ResponseWriter, r *http.Request, err error, protocol config.Protocol) {
	var statusCode int
	switch {
	case errors.Is(err, context.Canceled):
		statusCode = 499
	case errors.Is(err, context.DeadlineExceeded):
		statusCode = 504
	default:
		statusCode = 502
	}
	_metricRequestsTotol.WithLabelValues(protocol.String(), r.Method, r.URL.Path, strconv.Itoa(statusCode)).Inc()
	if protocol == config.Protocol_GRPC {
		// see https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
		code := strconv.Itoa(int(status.ToGRPCCode(statusCode)))
		w.Header().Set("Content-Type", "application/grpc")
		w.Header().Set("Grpc-Status", code)
		w.Header().Set("Grpc-Message", err.Error())
		statusCode = 200
	}
	w.WriteHeader(statusCode)
}

// Proxy is a gateway proxy.
type Proxy struct {
	router            atomic.Value
	clientFactory     client.Factory
	middlewareFactory middleware.Factory
	recoverStackPool  sync.Pool
}

// New is new a gateway proxy.
func New(clientFactory client.Factory, middlewareFactory middleware.Factory) (*Proxy, error) {
	p := &Proxy{
		clientFactory:     clientFactory,
		middlewareFactory: middlewareFactory,
	}
	p.recoverStackPool.New = func() interface{} {
		return make([]byte, 64<<10)
	}
	p.router.Store(mux.NewRouter())
	return p, nil
}

func (p *Proxy) buildMiddleware(ms []*config.Middleware, next http.RoundTripper) (http.RoundTripper, error) {
	for i := len(ms) - 1; i >= 0; i-- {
		m, err := p.middlewareFactory(ms[i])
		if err != nil {
			return nil, err
		}
		next = m(next)
	}
	return next, nil
}

func (p *Proxy) buildEndpoint(e *config.Endpoint, ms []*config.Middleware) (http.Handler, error) {
	tripper, err := p.clientFactory(e)
	if err != nil {
		return nil, err
	}
	tripper, err = p.buildMiddleware(e.Middlewares, tripper)
	if err != nil {
		return nil, err
	}
	tripper, err = p.buildMiddleware(ms, tripper)
	if err != nil {
		return nil, err
	}
	retryStrategy, err := prepareRetryStrategy(e)
	if err != nil {
		return nil, err
	}
	protocol := e.Protocol.String()
	return http.Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		startTime := time.Now()
		setXFFHeader(req)

		ctx := middleware.NewRequestContext(req.Context(), middleware.NewRequestOptions(e))
		ctx, cancel := context.WithTimeout(ctx, retryStrategy.timeout)
		defer cancel()
		defer func() {
			_metricRequestsDuration.WithLabelValues(protocol, req.Method, req.URL.Path).Observe(time.Since(startTime).Seconds())
		}()

		body, err := io.ReadAll(req.Body)
		if err != nil {
			writeError(w, req, err, e.Protocol)
			return
		}
		_metricReceivedBytes.WithLabelValues(protocol, req.Method, req.URL.Path).Add(float64(len(body)))
		req.GetBody = func() (io.ReadCloser, error) {
			reader := bytes.NewReader(body)
			return ioutil.NopCloser(reader), nil
		}

		var resp *http.Response
		for i := 0; i < int(retryStrategy.attempts); i++ {
			if i > 0 {
				_metricRetryTotal.WithLabelValues(protocol, req.Method, req.URL.Path).Inc()
			}
			// canceled or deadline exceeded
			if err = ctx.Err(); err != nil {
				break
			}
			tryCtx, cancel := context.WithTimeout(ctx, retryStrategy.perTryTimeout)
			defer cancel()
			reader := bytes.NewReader(body)
			req.Body = ioutil.NopCloser(reader)
			resp, err = tripper.RoundTrip(req.WithContext(tryCtx))
			if err != nil {
				LOG.Errorf("Attempt at [%d/%d], failed to handle request: %s: %+v", i+1, retryStrategy.attempts, req.URL.String(), err)
				continue
			}
			if !judgeRetryRequired(retryStrategy.conditions, resp) {
				if i > 0 {
					_metricRetrySuccess.WithLabelValues(protocol, req.Method, req.URL.Path).Inc()
				}
				break
			}
			// continue the retry loop
		}
		if err != nil {
			writeError(w, req, err, e.Protocol)
			return
		}

		headers := w.Header()
		for k, v := range resp.Header {
			headers[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		if body := resp.Body; body != nil {
			sent, err := io.Copy(w, body)
			if err != nil {
				LOG.Errorf("Failed to copy backend response body to client: [%s] %s %s %+v\n", e.Protocol, e.Method, e.Path, err)
			}
			_metricSentBytes.WithLabelValues(protocol, req.Method, req.URL.Path).Add(float64(sent))
		}
		// see https://pkg.go.dev/net/http#example-ResponseWriter-Trailers
		for k, v := range resp.Trailer {
			headers[http.TrailerPrefix+k] = v
		}
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
		_metricRequestsTotol.WithLabelValues(protocol, req.Method, req.URL.Path, "200").Inc()
	})), nil
}

// Update updates service endpoint.
func (p *Proxy) Update(c *config.Gateway) error {
	newRouter := mux.NewRouter()
	for _, e := range c.Endpoints {
		handler, err := p.buildEndpoint(e, c.Middlewares)
		if err != nil {
			return err
		}
		if err = newRouter.Handle(e.Path, e.Method, handler); err != nil {
			return err
		}
		LOG.Infof("build endpoint: [%s] %s %s", e.Protocol, e.Method, e.Path)
	}
	p.router.Store(newRouter)
	return nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			w.WriteHeader(http.StatusBadGateway)
			buf := p.recoverStackPool.Get().([]byte)
			n := runtime.Stack(buf, false)
			LOG.Errorf("panic recovered: %s", buf[:n])
			p.recoverStackPool.Put(buf)
		}
	}()
	p.router.Load().(router.Router).ServeHTTP(w, req)
}

// DebugHandler implemented debug handler.
func (p *Proxy) DebugHandler() http.Handler {
	debugMux := gorillamux.NewRouter()
	debugMux.Methods("GET").Path("/debug/proxy/router/inspect").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		loadRouter, ok := p.router.Load().(router.Router)
		if !ok {
			return
		}
		inspect := mux.InspectMuxRouter(loadRouter)
		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(inspect)
	})
	return debugMux
}
