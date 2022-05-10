package circuitbreaker

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"net/http"
	"sync"

	"github.com/go-kratos/aegis/circuitbreaker"
	"github.com/go-kratos/aegis/circuitbreaker/sre"
	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/circuitbreaker/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/proxy/condition"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func Init(clientFactory client.Factory) {
	breakerFactory := New(clientFactory)
	middleware.Register("circuitbreaker", breakerFactory)
}

var (
	LOG                = log.NewHelper(log.With(log.GetLogger(), "source", "accesslog"))
	_metricDeniedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go",
		Subsystem: "gateway",
		Name:      "requests_circuit_breaker_denied_total",
		Help:      "The total number of denied requests",
	}, []string{"method", "path"})
)

type ratioTrigger struct {
	*v1.CircuitBreaker_Ratio
	lock sync.Mutex
	rand *rand.Rand
}

func newRatioTrigger(in *v1.CircuitBreaker_Ratio) *ratioTrigger {
	return &ratioTrigger{
		CircuitBreaker_Ratio: in,
		rand:                 rand.New(rand.NewSource(rand.Int63())),
	}
}

func (r *ratioTrigger) Allow() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.rand.Int63n(10000) < r.Ratio {
		return nil
	}
	return circuitbreaker.ErrNotAllowed
}
func (*ratioTrigger) MarkSuccess() {}
func (*ratioTrigger) MarkFailed()  {}

type nopTrigger struct{}

func (nopTrigger) Allow() error { return nil }
func (nopTrigger) MarkSuccess() {}
func (nopTrigger) MarkFailed()  {}

func makeBreakerTrigger(in *v1.CircuitBreaker) circuitbreaker.CircuitBreaker {
	switch trigger := in.Trigger.(type) {
	case *v1.CircuitBreaker_SuccessRatio:
		opts := []sre.Option{}
		if trigger.SuccessRatio.Bucket != 0 {
			opts = append(opts, sre.WithBucket(int(trigger.SuccessRatio.Bucket)))
		}
		if trigger.SuccessRatio.Request != 0 {
			opts = append(opts, sre.WithRequest(int64(trigger.SuccessRatio.Request)))
		}
		if trigger.SuccessRatio.Success != 0 {
			opts = append(opts, sre.WithSuccess(trigger.SuccessRatio.Success))
		}
		if trigger.SuccessRatio.Window != nil {
			opts = append(opts, sre.WithWindow(trigger.SuccessRatio.Window.AsDuration()))
		}
		return sre.NewBreaker(opts...)
	case *v1.CircuitBreaker_Ratio:
		return newRatioTrigger(trigger)
	default:
		LOG.Warnf("Unrecoginzed circuit breaker trigger: %+v", trigger)
		return nopTrigger{}
	}
}

func makeOnBreakHandler(in *v1.CircuitBreaker, factory client.Factory) (middleware.Handler, error) {
	switch action := in.Action.(type) {
	case *v1.CircuitBreaker_BackupService:
		LOG.Infof("Making backup service as on break handler: %+v", action)
		client, err := factory(action.BackupService.Endpoint)
		if err != nil {
			return nil, err
		}
		return client.Do, nil
	case *v1.CircuitBreaker_ResponseData:
		LOG.Infof("Making static response data as on break handler: %+v", action)
		body := io.NopCloser(bytes.NewBuffer(action.ResponseData.Body))
		resp := &http.Response{
			StatusCode: int(action.ResponseData.StatusCode),
			Header:     http.Header{},
			Body:       body,
		}
		for _, h := range action.ResponseData.Header {
			resp.Header[h.Key] = h.Value
		}
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			return resp, nil
		}, nil
	default:
		LOG.Warnf("Unrecoginzed circuit breaker aciton: %+v", action)
		return func(context.Context, *http.Request) (*http.Response, error) {
			// TBD: on break response
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Header:     http.Header{},
			}, nil
		}, nil
	}
}

func isSuccessResponse(conditions []condition.Condition, resp *http.Response) bool {
	return condition.JudgeConditons(conditions, resp, true)
}

func New(factory client.Factory) middleware.Factory {
	return middleware.Factory(func(c *config.Middleware) (middleware.Middleware, error) {
		options := &v1.CircuitBreaker{}
		if c.Options != nil {
			if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
				return nil, err
			}
		}
		breaker := makeBreakerTrigger(options)
		onBreakHandler, err := makeOnBreakHandler(options, factory)
		if err != nil {
			return nil, err
		}
		assertCondtions, err := condition.ParseConditon(options.AssertCondtions...)
		if err != nil {
			return nil, err
		}
		return func(handler middleware.Handler) middleware.Handler {
			return func(ctx context.Context, req *http.Request) (*http.Response, error) {
				if err := breaker.Allow(); err != nil {
					// rejected
					// NOTE: when client reject requets locally,
					// continue add counter let the drop ratio higher.
					breaker.MarkFailed()
					_metricDeniedTotal.WithLabelValues(req.Method, req.URL.Path).Inc()
					return onBreakHandler(ctx, req)
				}
				resp, err := handler(ctx, req)
				if err != nil {
					breaker.MarkFailed()
					return onBreakHandler(ctx, req)
				}
				if !isSuccessResponse(assertCondtions, resp) {
					breaker.MarkFailed()
					return onBreakHandler(ctx, req)
				}
				breaker.MarkSuccess()
				return resp, nil
			}
		}, nil
	})
}
