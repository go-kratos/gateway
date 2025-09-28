package streamrecorder

import (
	"fmt"
	"io"
	"net/http"
	"sync/atomic"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

func init() {
	middleware.RegisterV2("streamrecorder", New)
}

func New(*configv1.Middleware) (middleware.MiddlewareV2, error) {
	return &streamRecorderWrapper{}, nil
}

type streamRecorderWrapper struct{}

var _ middleware.MiddlewareV2 = (*streamRecorderWrapper)(nil)

func (s *streamRecorderWrapper) Process(next http.RoundTripper) http.RoundTripper {
	return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		reply, err := next.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if reply.ProtoMajor == 2 {
			s.processH2(req, reply)
			return reply, nil
		}
		s.processH1(req, reply)
		return reply, nil
	})
}

func (s *streamRecorderWrapper) processH1(_ *http.Request, reply *http.Response) {
	if reply.Body != nil {
		rwc, ok := reply.Body.(io.ReadWriteCloser)
		if ok {
			w := newReadWriteCloserBody(rwc)
			go func() {
				<-w.done
				fmt.Println("WEBSOCKET PROXY DONE")
			}()
			reply.Body = w
		}
	}
}

func (s *streamRecorderWrapper) processH2(req *http.Request, reply *http.Response) {
	messages := []message{}
	if req.Body != nil {
		w := newReadCloserBody(req.Body, tagRequest, messages)
		req.Body = w
	}
	if reply.Body != nil {
		w := newReadCloserBody(reply.Body, tagResponse, messages)
		go func() {
			<-w.done
			fmt.Println("GRPC STREAM PROXY DONE")
		}()
		reply.Body = w
	}
}

func (s *streamRecorderWrapper) Close() error {
	return nil
}

const (
	tagRead  = 0
	tagWrite = 1

	tagRequest  = 0
	tagResponse = 1
)

type message struct {
	idx  int64
	tag  int8
	data []byte
}

var _ http.CloseNotifier = (*readWriteCloserBody)(nil)

type readWriteCloserBody struct {
	idx      int64
	messages struct {
		read  []message
		write []message
	}
	done chan bool
	io.ReadWriteCloser
}

func newReadWriteCloserBody(rwc io.ReadWriteCloser) *readWriteCloserBody {
	return &readWriteCloserBody{
		done: make(chan bool),
		messages: struct {
			read  []message
			write []message
		}{
			read:  []message{},
			write: []message{},
		},
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

func (b *readWriteCloserBody) Read(p []byte) (n int, err error) {
	idx := atomic.AddInt64(&b.idx, 1)
	b.messages.read = append(b.messages.read, message{idx: idx, tag: tagRead, data: p})
	return b.ReadWriteCloser.Read(p)
}

func (b *readWriteCloserBody) Write(p []byte) (n int, err error) {
	idx := atomic.AddInt64(&b.idx, 1)
	b.messages.write = append(b.messages.write, message{idx: idx, tag: tagWrite, data: p})
	return b.ReadWriteCloser.Write(p)
}

func (b *readWriteCloserBody) String() string {
	return fmt.Sprintf("readWriteCloserBody{idx: %d, messages: %d}", b.idx, len(b.messages.read)+len(b.messages.write))
}

var _ http.CloseNotifier = (*readCloserBody)(nil)

type readCloserBody struct {
	idx      int64
	tag      int8
	messages []message
	done     chan bool
	io.ReadCloser
}

func newReadCloserBody(rc io.ReadCloser, tag int8, messages []message) *readCloserBody {
	return &readCloserBody{
		done:       make(chan bool),
		messages:   messages,
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

func (b *readCloserBody) Read(p []byte) (n int, err error) {
	idx := atomic.AddInt64(&b.idx, 1)
	b.messages = append(b.messages, message{idx: idx, tag: b.tag, data: p})
	return b.ReadCloser.Read(p)
}

func (b *readCloserBody) String() string {
	return fmt.Sprintf("readCloserBody{idx: %d, tag: %d, messages: %v}", b.idx, b.tag, b.messages)
}
