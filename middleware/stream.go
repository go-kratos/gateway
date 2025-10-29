package middleware

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
)

type StreamBody interface {
	CloseNotify() <-chan bool
}

const (
	TagRequest  = "request"
	TagResponse = "response"
)

type MetaStreamContextKey struct{}
type MetaStreamContext struct {
	Request    *http.Request
	Response   *http.Response
	OnResponse []func(req *http.Request, reply *http.Response)
	OnFinish   []func(req *http.Request, reply *http.Response)
	OnChunk    []func(req *http.Request, reply *http.Response, chunk *MetaStreamChunk)

	// For bidirectional streaming: track when both request and response bodies are closed
	// bodiesCount is the number of bodies to wait for (0, 1, or 2)
	// closedCount tracks how many have been closed
	bodiesCount int32
	closedCount int32
	finishOnce  sync.Once
}

func (s *MetaStreamContext) DoOnResponse() {
	for _, fn := range s.OnResponse {
		fn(s.Request, s.Response)
	}
}

// RegisterBody increments the count of bodies to wait for before calling OnFinish
func (s *MetaStreamContext) RegisterBody() {
	atomic.AddInt32(&s.bodiesCount, 1)
}

// notifyBodyClosed is called when a body is closed. Returns true if all bodies are closed.
func (s *MetaStreamContext) notifyBodyClosed() bool {
	closed := atomic.AddInt32(&s.closedCount, 1)
	expected := atomic.LoadInt32(&s.bodiesCount)
	return closed >= expected && expected > 0
}

func InitMetaStreamContext(opts *RequestOptions, value *MetaStreamContext) {
	opts.Values.Set(MetaStreamContextKey{}, value)
}

func GetMetaStreamContext(opts *RequestOptions) (*MetaStreamContext, bool) {
	value, ok := opts.Values.Get(MetaStreamContextKey{})
	if !ok {
		return nil, false
	}
	return value.(*MetaStreamContext), true
}

type MetaStreamChunk struct {
	Tag  string
	Data []byte
	Err  error
}

var _ StreamBody = (*readWriteCloserBody)(nil)

type readWriteCloserBody struct {
	ctxValue *MetaStreamContext
	doneOnce sync.Once
	done     chan bool
	io.ReadWriteCloser
}

func WrapReadWriteCloserBody(rwc io.ReadWriteCloser, ctxValue *MetaStreamContext) *readWriteCloserBody {
	ctxValue.RegisterBody()
	return &readWriteCloserBody{
		ctxValue:        ctxValue,
		done:            make(chan bool),
		ReadWriteCloser: rwc,
	}
}

func (b *readWriteCloserBody) CloseNotify() <-chan bool {
	return b.done
}

func (b *readWriteCloserBody) Close() error {
	b.doneOnce.Do(func() {
		close(b.done)
		// Only call OnFinish when all registered bodies are closed
		if b.ctxValue.notifyBodyClosed() {
			b.ctxValue.finishOnce.Do(func() {
				for _, fn := range b.ctxValue.OnFinish {
					fn(b.ctxValue.Request, b.ctxValue.Response)
				}
			})
		}
	})
	return b.ReadWriteCloser.Close()
}

func (b *readWriteCloserBody) Read(p []byte) (int, error) {
	n, err := b.ReadWriteCloser.Read(p)
	m := &MetaStreamChunk{Tag: TagResponse, Data: bytes.Clone(p[:n]), Err: err}
	defer func() {
		for _, fn := range b.ctxValue.OnChunk {
			fn(b.ctxValue.Request, b.ctxValue.Response, m)
		}
	}()
	return n, err
}

func (b *readWriteCloserBody) Write(p []byte) (int, error) {
	n, err := b.ReadWriteCloser.Write(p)
	m := &MetaStreamChunk{Tag: TagRequest, Data: bytes.Clone(p[:n]), Err: err}
	defer func() {
		for _, fn := range b.ctxValue.OnChunk {
			fn(b.ctxValue.Request, b.ctxValue.Response, m)
		}
	}()
	return n, err
}

var _ StreamBody = (*readCloserBody)(nil)

type readCloserBody struct {
	ctxValue *MetaStreamContext
	tag      string
	doneOnce sync.Once
	done     chan bool
	io.ReadCloser
}

func WrapReadCloserBody(rc io.ReadCloser, tag string, ctxValue *MetaStreamContext) *readCloserBody {
	ctxValue.RegisterBody()
	return &readCloserBody{
		ctxValue:   ctxValue,
		done:       make(chan bool),
		tag:        tag,
		ReadCloser: rc,
	}
}

func (b *readCloserBody) CloseNotify() <-chan bool {
	return b.done
}

func (b *readCloserBody) Close() error {
	// In reverse proxy, the body maybe closed multiple times, so we need to use a sync.Once to ensure it is closed only once.
	b.doneOnce.Do(func() {
		close(b.done)
		// Only call OnFinish when all registered bodies (request + response) are closed
		if b.ctxValue.notifyBodyClosed() {
			b.ctxValue.finishOnce.Do(func() {
				for _, fn := range b.ctxValue.OnFinish {
					fn(b.ctxValue.Request, b.ctxValue.Response)
				}
			})
		}
	})
	return b.ReadCloser.Close()
}

func (b *readCloserBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	m := &MetaStreamChunk{Tag: b.tag, Data: bytes.Clone(p[:n]), Err: err}
	defer func() {
		for _, fn := range b.ctxValue.OnChunk {
			fn(b.ctxValue.Request, b.ctxValue.Response, m)
		}
	}()
	return n, err
}
