package endpoint

import (
	"io"
	"net/http"
	"net/url"
	"sync"
)

var reqPool = &sync.Pool{
	New: func() interface{} {
		return new(httpRequest)
	},
}

// FreeRequest free request object.
func FreeRequest(req Request) {
	if r, ok := req.(*httpRequest); ok {
		r.reset(nil)
		reqPool.Put(r)
	}
}

type httpRequest struct {
	*http.Request
}

// NewRequest new an HTTP request.
func NewRequest(req *http.Request) Request {
	r := reqPool.Get().(*httpRequest)
	r.reset(req)
	return r
}

func (r *httpRequest) reset(req *http.Request) {
	r.Request = req
}

func (r *httpRequest) Host() string {
	return r.Request.Host
}

func (r *httpRequest) Path() string {
	return r.Request.RequestURI
}
func (r *httpRequest) Method() string {
	return r.Request.Method
}
func (r *httpRequest) Query() url.Values {
	return r.Request.URL.Query()
}
func (r *httpRequest) Header() http.Header {
	return r.Request.Header
}
func (r *httpRequest) Body() io.ReadCloser {
	return r.Request.Body
}
