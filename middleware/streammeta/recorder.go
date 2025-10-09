package streammeta

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

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
	ctxValue *MetaStreamContextValue
	chunks   *bodyChunks
	done     chan bool
	io.ReadWriteCloser
}

func newReadWriteCloserBody(rwc io.ReadWriteCloser, ctxValue *MetaStreamContextValue) *readWriteCloserBody {
	return &readWriteCloserBody{
		ctxValue:        ctxValue,
		done:            make(chan bool),
		chunks:          newBodyChunks(),
		ReadWriteCloser: rwc,
	}
}

func (b *readWriteCloserBody) CloseNotify() <-chan bool {
	return b.done
}

func (b *readWriteCloserBody) Close() error {
	defer func() {
		for _, fn := range b.ctxValue.OnFinish {
			fn(b.ctxValue.Request, b.ctxValue.Response)
		}
	}()
	close(b.done)
	return b.ReadWriteCloser.Close()
}

func (b *readWriteCloserBody) Read(p []byte) (int, error) {
	n, err := b.ReadWriteCloser.Read(p)
	m := &message{Tag: TagResponse, Data: bytes.Clone(p[:n])}
	b.chunks.Append(m)
	defer func() {
		for _, fn := range b.ctxValue.OnChunk {
			fn(b.ctxValue.Request, b.ctxValue.Response, m)
		}
	}()
	return n, err
}

func (b *readWriteCloserBody) Write(p []byte) (int, error) {
	n, err := b.ReadWriteCloser.Write(p)
	m := &message{Tag: TagRequest, Data: bytes.Clone(p[:n])}
	b.chunks.Append(m)
	defer func() {
		for _, fn := range b.ctxValue.OnChunk {
			fn(b.ctxValue.Request, b.ctxValue.Response, m)
		}
	}()
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
	ctxValue *MetaStreamContextValue
	tag      string
	chunks   *bodyChunks
	done     chan bool
	io.ReadCloser
}

func newReadCloserBody(rc io.ReadCloser, tag string, chunks *bodyChunks, ctxValue *MetaStreamContextValue) *readCloserBody {
	return &readCloserBody{
		ctxValue:   ctxValue,
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
	defer func() {
		for _, fn := range b.ctxValue.OnFinish {
			fn(b.ctxValue.Request, b.ctxValue.Response)
		}
	}()
	close(b.done)
	return b.ReadCloser.Close()
}

func (b *readCloserBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	m := &message{Tag: b.tag, Data: bytes.Clone(p[:n])}
	b.chunks.Append(m)
	defer func() {
		for _, fn := range b.ctxValue.OnChunk {
			fn(b.ctxValue.Request, b.ctxValue.Response, m)
		}
	}()
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

func ResponseBytes(in []*message) []byte {
	var buf bytes.Buffer
	for _, m := range in {
		if m.Tag == TagResponse {
			buf.Write(m.Data)
		}
	}
	return buf.Bytes()
}

func RequestBytes(in []*message) []byte {
	var buf bytes.Buffer
	for _, m := range in {
		if m.Tag == TagRequest {
			buf.Write(m.Data)
		}
	}
	return buf.Bytes()
}
