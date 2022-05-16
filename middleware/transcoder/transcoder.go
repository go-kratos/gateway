package transcoder

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func decodeBinHeader(v string) ([]byte, error) {
	if len(v)%4 == 0 {
		// Input was padded, or padding was not necessary.
		return base64.StdEncoding.DecodeString(v)
	}
	return base64.RawStdEncoding.DecodeString(v)
}

func newResponse(statusCode int, header http.Header, data []byte) (*http.Response, error) {
	return &http.Response{
		Header:        header,
		StatusCode:    statusCode,
		ContentLength: int64(len(data)),
		Body:          ioutil.NopCloser(bytes.NewReader(data)),
	}, nil
}

func init() {
	middleware.Register("transcoder", Middleware)
}

// Middleware is a gRPC transcoder.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			ctx := req.Context()
			contentType := req.Header.Get("Content-Type")
			endpoint, _ := middleware.EndpointFromContext(ctx)
			if endpoint.Protocol != config.Protocol_GRPC || strings.HasPrefix(contentType, "application/grpc") {
				return next.RoundTrip(req)
			}
			b, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			bb := make([]byte, len(b)+5)
			binary.BigEndian.PutUint32(bb[1:], uint32(len(b)))
			copy(bb[5:], b)
			// content-type:
			// - application/grpc+json
			// - application/grpc+proto
			req.Header.Set("Content-Type", "application/grpc+"+strings.TrimLeft(contentType, "application/"))
			req.Header.Del("Content-Length")
			req.ContentLength = int64(len(bb))
			req.Body = ioutil.NopCloser(bytes.NewReader(bb))
			resp, err := next.RoundTrip(req)
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			// Convert HTTP/2 response to HTTP/1.1
			// Trailers are sent in a data frame, so don't announce trailers as otherwise downstream proxies might get confused.
			for trailerName, values := range resp.Trailer {
				resp.Header[trailerName] = values
			}
			resp.Trailer = nil
			resp.Header.Set("Content-Type", contentType)
			if grpcStatus := resp.Header.Get("grpc-status"); grpcStatus != "0" {
				code, err := strconv.ParseInt(grpcStatus, 10, 64)
				if err != nil {
					return nil, err
				}
				st := &spb.Status{
					Code:    int32(code),
					Message: resp.Header.Get("grpc-message"),
				}
				if grpcDetails := resp.Header.Get("grpc-status-details-bin"); grpcDetails != "" {
					details, err := decodeBinHeader(grpcDetails)
					if err != nil {
						return nil, err
					}
					if err = proto.Unmarshal(details, st); err != nil {
						return nil, err
					}
				}
				data, err := protojson.Marshal(st)
				if err != nil {
					return nil, err
				}
				return newResponse(200, resp.Header, data)
			}
			resp.Body = ioutil.NopCloser(bytes.NewReader(data[5:]))
			resp.ContentLength = int64(len(data) - 5)
			// Any content length that might be set is no longer accurate because of trailers.
			resp.Header.Del("Content-Length")
			return resp, nil
		})
	}, nil
}
