package streamrecorder

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

func init() {
	middleware.RegisterV2("streamrecorder", New)
}

func New(*configv1.Middleware) (middleware.MiddlewareV2, error) {
	return &streamRecorder{}, nil
}

type streamRecorder struct{}

var _ middleware.MiddlewareV2 = (*streamRecorder)(nil)

func (s *streamRecorder) Process(next http.RoundTripper) http.RoundTripper {
	return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.ProtoMajor == 2 {
			chunks := newBodyChunks()
			if req.Body != nil {
				w := newReadCloserBody(req.Body, TagRequest, chunks)
				req.Body = w
			}
			reply, err := next.RoundTrip(req)
			if err != nil {
				return nil, err
			}
			s.processH2(req, reply, chunks)
			return reply, nil
		}

		reply, err := next.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		s.processH1(req, reply)
		return reply, nil
	})
}

func (s *streamRecorder) processH1(_ *http.Request, reply *http.Response) {
	if reply.Body != nil {
		rwc, ok := reply.Body.(io.ReadWriteCloser)
		if ok {
			w := newReadWriteCloserBody(rwc)
			reply.Body = w
			return
		}
		w := newReadCloserBody(reply.Body, TagResponse, newBodyChunks())
		reply.Body = w
		return
	}
}

func (s *streamRecorder) processH2(_ *http.Request, reply *http.Response, chunks *bodyChunks) {
	if reply.Body != nil {
		w := newReadCloserBody(reply.Body, TagResponse, chunks)
		reply.Body = w
	}
}

func (s *streamRecorder) Close() error {
	return nil
}

const (
	TagRequest  = "request"
	TagResponse = "response"
)

type message struct {
	Index int64
	Tag   string
	Data  []byte
}

var _ ChunkRecorder = (*readWriteCloserBody)(nil)

type readWriteCloserBody struct {
	chunks *bodyChunks
	done   chan bool
	io.ReadWriteCloser
}

func newReadWriteCloserBody(rwc io.ReadWriteCloser) *readWriteCloserBody {
	return &readWriteCloserBody{
		done:            make(chan bool),
		chunks:          newBodyChunks(),
		ReadWriteCloser: rwc,
	}
}

func (b *readWriteCloserBody) CloseNotify() <-chan bool {
	return b.done
}

func (b *readWriteCloserBody) Close() error {
	close(b.done)
	return b.ReadWriteCloser.Close()
}

func (b *readWriteCloserBody) Read(p []byte) (int, error) {
	n, err := b.ReadWriteCloser.Read(p)
	b.chunks.Append(&message{Tag: TagResponse, Data: bytes.Clone(p[:n])})
	return n, err
}

func (b *readWriteCloserBody) Write(p []byte) (int, error) {
	n, err := b.ReadWriteCloser.Write(p)
	b.chunks.Append(&message{Tag: TagRequest, Data: bytes.Clone(p[:n])})
	return n, err
}

func (b *readWriteCloserBody) String() string {
	return fmt.Sprintf("websocketIO{chunks: %d}", b.chunks.Len())
}

func (b *readWriteCloserBody) GetChunks() []*message {
	return b.chunks.GetChunks()
}

var _ ChunkRecorder = (*readCloserBody)(nil)

type bodyChunks struct {
	sync.Mutex
	length int64
	chunks []*message
}

func (c *bodyChunks) Append(m *message) {
	c.Lock()
	defer c.Unlock()
	m.Index = atomic.AddInt64(&c.length, 1)
	c.chunks = append(c.chunks, m)
}

func (c *bodyChunks) Len() int64 {
	return atomic.LoadInt64(&c.length)
}

func (c *bodyChunks) GetChunks() []*message {
	c.Lock()
	defer c.Unlock()
	out := make([]*message, c.length)
	copy(out, c.chunks)
	return out
}

func newBodyChunks() *bodyChunks {
	return &bodyChunks{
		chunks: make([]*message, 0),
		length: 0,
	}
}

type readCloserBody struct {
	tag    string
	chunks *bodyChunks
	done   chan bool
	io.ReadCloser
}

func newReadCloserBody(rc io.ReadCloser, tag string, chunks *bodyChunks) *readCloserBody {
	return &readCloserBody{
		done:       make(chan bool),
		chunks:     chunks,
		tag:        tag,
		ReadCloser: rc,
	}
}

func (b *readCloserBody) CloseNotify() <-chan bool {
	return b.done
}

func (b *readCloserBody) Close() error {
	close(b.done)
	return b.ReadCloser.Close()
}

func (b *readCloserBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.chunks.Append(&message{Tag: b.tag, Data: bytes.Clone(p[:n])})
	return n, err
}

func (b *readCloserBody) String() string {
	return fmt.Sprintf("http2Body{chunks: %d}", b.chunks.Len())
}

func (b *readCloserBody) GetChunks() []*message {
	return b.chunks.GetChunks()
}

type ChunkRecorder interface {
	CloseNotify() <-chan bool
	GetChunks() []*message
}
