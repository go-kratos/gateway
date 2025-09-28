package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"

	"github.com/go-kratos/aegis/circuitbreaker/sre"
	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/middleware"
)

func (p *Proxy) buildStreamEndpoint(buildContext *client.BuildContext, e *config.Endpoint, ms []*config.Middleware) (_ http.Handler, _ io.Closer, retError error) {
	client, err := p.clientFactory(buildContext, e)
	if err != nil {
		return nil, nil, err
	}
	tripper := http.RoundTripper(client)
	closer := io.Closer(client)
	defer closeOnError(closer, &retError)
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
	markSuccessStat, markFailedStat := splitRetryMetricsHandler(e)
	retryBreaker := sre.NewBreaker(sre.WithSuccess(0.8))
	markSuccess := func(req *http.Request, i int) {
		markSuccessStat(req, i)
		if i > 0 {
			retryBreaker.MarkSuccess()
		}
	}
	markFailed := func(req *http.Request, i int, err error) {
		markFailedStat(req, i, err)
		if i > 0 {
			retryBreaker.MarkFailed()
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		reqOpts := middleware.NewRequestOptions(e)
		// stream should always be last attempt
		reqOpts.LastAttempt = true
		ctx := middleware.NewRequestContext(req.Context(), reqOpts)
		ctx, cancel := context.WithTimeout(ctx, retryStrategy.timeout)
		defer cancel()

		reverseProxy := &httputil.ReverseProxy{
			Rewrite: func(_ *httputil.ProxyRequest) {},
			ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
				markFailed(req, 0, err)
				writeError(w, req, err, labels)
			},
			ModifyResponse: func(res *http.Response) error {
				markSuccess(req, 0)
				return nil
			},
			Transport:     tripper,
			FlushInterval: -1,
		}
		reverseProxy.ServeHTTP(w, req.Clone(ctx))
		fmt.Println("STREAM PROXY DONE")
	}), closer, nil
}
