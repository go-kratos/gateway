package middleware

import (
	"bytes"
	"io"
	"net/http"
	"sync"
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
	OnFinish   []func(req *http.Request, reply *http.Response, tag string)
	OnChunk    []func(req *http.Request, reply *http.Response, chunk *MetaStreamChunk)
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
	Index int64
	Tag   string
	Data  []byte
}

var _ StreamBody = (*readWriteCloserBody)(nil)

type readWriteCloserBody struct {
	ctxValue *MetaStreamContext
	doneOnce sync.Once
	done     chan bool
	io.ReadWriteCloser
}

func WrapReadWriteCloserBody(rwc io.ReadWriteCloser, ctxValue *MetaStreamContext) *readWriteCloserBody {
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
		defer func() {
			for _, fn := range b.ctxValue.OnFinish {
				fn(b.ctxValue.Request, b.ctxValue.Response, "")
			}
		}()
		close(b.done)
	})
	return b.ReadWriteCloser.Close()
}

func (b *readWriteCloserBody) Read(p []byte) (int, error) {
	n, err := b.ReadWriteCloser.Read(p)
	m := &MetaStreamChunk{Tag: TagResponse, Data: bytes.Clone(p[:n])}
	defer func() {
		for _, fn := range b.ctxValue.OnChunk {
			fn(b.ctxValue.Request, b.ctxValue.Response, m)
		}
	}()
	return n, err
}

func (b *readWriteCloserBody) Write(p []byte) (int, error) {
	n, err := b.ReadWriteCloser.Write(p)
	m := &MetaStreamChunk{Tag: TagRequest, Data: bytes.Clone(p[:n])}
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
		defer func() {
			for _, fn := range b.ctxValue.OnFinish {
				fn(b.ctxValue.Request, b.ctxValue.Response, b.tag)
			}
		}()
		close(b.done)
	})
	return b.ReadCloser.Close()
}

func (b *readCloserBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	m := &MetaStreamChunk{Tag: b.tag, Data: bytes.Clone(p[:n])}
	defer func() {
		for _, fn := range b.ctxValue.OnChunk {
			fn(b.ctxValue.Request, b.ctxValue.Response, m)
		}
	}()
	return n, err
}
