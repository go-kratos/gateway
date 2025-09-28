package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"

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
	// retryStrategy, err := prepareRetryStrategy(e)
	// if err != nil {
	// 	return nil, nil, err
	// }
	return &httputil.ReverseProxy{
		Rewrite: func(proxy *httputil.ProxyRequest) {
			reqOpts := middleware.NewRequestOptions(e)
			ctx := middleware.NewRequestContext(proxy.Out.Context(), reqOpts)
			newReq := proxy.Out.WithContext(ctx)
			proxy.Out = newReq
			fmt.Println("Incoming request", proxy.Out.URL.String())
		},
		ModifyResponse: func(res *http.Response) error {
			fmt.Println("Outgoing response", res.StatusCode, res.Header)
			return nil
		},
		Transport:     tripper,
		FlushInterval: -1,
	}, closer, nil
}
