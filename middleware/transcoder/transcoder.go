package transcoder

import (
	"context"
	"net/http"
	"strings"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/proxy"
)

const (
	forceRetryStatus = http.StatusInternalServerError
)

func init() {
	middleware.Register("grpc_transcoder", Middleware)
}

// Middleware is a gRPC transcoder.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			contentType := req.Header.Get("Content-Type")
			endpoint, _ := middleware.EndpointFromContext(ctx)
			if endpoint.Protocol != config.Protocol_GRPC || strings.HasPrefix(contentType, "application/grpc") {
				return handler(ctx, req)
			}

			body := &proxy.BodyReader{}
			n, err := body.EncodeGRPC(req.Body)
			if err != nil {
				return nil, err
			}
			// content-type:
			// - application/grpc+json
			// - application/grpc+proto
			req.Header.Set("Content-Type", "application/grpc+"+strings.TrimLeft(contentType, "application/"))
			req.Header.Del("Content-Length")
			req.ContentLength = n
			req.Body = body
			resp, err := handler(ctx, req)
			if err != nil {
				return nil, err
			}
			if _, err := body.ReadFrom(resp.Body); err != nil {
				return nil, err
			}
			// skip header length of the gRPC body
			if _, err := body.Seek(5, 0); err != nil {
				return nil, err
			}
			resp.Body = body
			resp.Header.Set("Content-Type", contentType)
			// Convert HTTP/2 response to HTTP/1.1
			// Trailers are sent in a data frame, so don't announce trailers as otherwise downstream proxies might get confused.
			for trailerName, values := range resp.Trailer {
				resp.Header[trailerName] = values
			}
			resp.Trailer = nil
			// Any content length that might be set is no longer accurate because of trailers.
			resp.Header.Del("Content-Length")
			return resp, nil
		}
	}, nil
}
