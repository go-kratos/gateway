package transcoder

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/proxy"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	forceRetryStatus = http.StatusInternalServerError
)

func decodeBinHeader(v string) ([]byte, error) {
	if len(v)%4 == 0 {
		// Input was padded, or padding was not necessary.
		return base64.StdEncoding.DecodeString(v)
	}
	return base64.RawStdEncoding.DecodeString(v)
}

func decodeStatusDetails(rawDetails string) error {
	v, err := decodeBinHeader(rawDetails)
	if err != nil {
		return err
	}
	st := &spb.Status{}
	if err = proto.Unmarshal(v, st); err != nil {
		return err
	}
	return status.ErrorProto(st)
}

func decodeError(resp *http.Response) error {
	var err error
	if v := resp.Header.Get("grpc-status-details-bin"); v != "" {
		err = decodeStatusDetails(v)
	}
	return err
}

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
			contentLength, err := body.ReadFrom(resp.Body)
			if err != nil {
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
			if contentLength >= 5 {
				resp.ContentLength = contentLength - 5
			}
			if resp.Header.Get("grpc-status") != "0" {
				if err := decodeError(resp); err != nil {
					return nil, err
				}
			}
			return resp, nil
		}
	}, nil
}
