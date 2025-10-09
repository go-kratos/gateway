package streammeta

import (
	"context"
	"io"
	"net/http"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

func init() {
	middleware.RegisterV2("streamrecorder", New)
}

func New(*configv1.Middleware) (middleware.MiddlewareV2, error) {
	return &MetaStreamRecorder{}, nil
}

type MetaStreamContextKey struct{}
type MetaStreamContextValue struct {
	Request  *http.Request
	Response *http.Response
	Recorder *MetaStreamRecorder
	OnFinish []func(req *http.Request, reply *http.Response)
	OnChunk  []func(req *http.Request, reply *http.Response, chunk *message)
}

func NewContext(ctx context.Context, value *MetaStreamContextValue) context.Context {
	return context.WithValue(ctx, MetaStreamContextKey{}, value)
}

func FromContext(ctx context.Context) (*MetaStreamContextValue, bool) {
	value, ok := ctx.Value(MetaStreamContextKey{}).(*MetaStreamContextValue)
	if !ok {
		return nil, false
	}
	return value, true
}

type MetaStreamRecorder struct {
}

var _ middleware.MiddlewareV2 = (*MetaStreamRecorder)(nil)

func NewMetaStreamRecorder() *MetaStreamRecorder {
	return &MetaStreamRecorder{}
}

func (s *MetaStreamRecorder) Process(next http.RoundTripper) http.RoundTripper {
	return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		ctxValue, ok := FromContext(req.Context())
		if !ok {
			ctxValue = &MetaStreamContextValue{
				Request:  req,
				Response: nil,
				Recorder: s,
				OnFinish: nil,
				OnChunk:  nil,
			}
			ctx := NewContext(req.Context(), ctxValue)
			req = req.WithContext(ctx)
		}

		if req.ProtoMajor == 2 {
			chunks := newBodyChunks()
			if req.Body != nil {
				w := newReadCloserBody(req.Body, TagRequest, chunks, ctxValue)
				req.Body = w
			}
			reply, err := next.RoundTrip(req)
			if err != nil {
				return nil, err
			}
			ctxValue.Response = reply
			s.processH2(req, reply, chunks, ctxValue)
			return reply, nil
		}

		reply, err := next.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		ctxValue.Response = reply
		s.processH1(req, reply, ctxValue)
		return reply, nil
	})
}

func (s *MetaStreamRecorder) processH1(_ *http.Request, reply *http.Response, ctxValue *MetaStreamContextValue) {
	if reply.Body != nil {
		rwc, ok := reply.Body.(io.ReadWriteCloser)
		if ok {
			w := newReadWriteCloserBody(rwc, ctxValue)
			reply.Body = w
			return
		}
		w := newReadCloserBody(reply.Body, TagResponse, newBodyChunks(), ctxValue)
		reply.Body = w
		return
	}
}

func (s *MetaStreamRecorder) processH2(_ *http.Request, reply *http.Response, chunks *bodyChunks, ctxValue *MetaStreamContextValue) {
	if reply.Body != nil {
		w := newReadCloserBody(reply.Body, TagResponse, chunks, ctxValue)
		reply.Body = w
	}
}

func (s *MetaStreamRecorder) Close() error {
	return nil
}
