package endpoint

import (
	"io"
	"net/http"
	"sync"
)

var resPool = &sync.Pool{
	New: func() interface{} {
		return new(httpResponse)
	},
}

// FreeResponse free response object.
func FreeResponse(res Response) {
	if r, ok := res.(*httpResponse); ok {
		r.reset(nil)
		resPool.Put(r)
	}
}

type httpResponse struct {
	*http.Response
}

// NewResponse new an HTTP response.
func NewResponse(res *http.Response) Response {
	r := resPool.Get().(*httpResponse)
	r.reset(res)
	return r
}

func (r *httpResponse) reset(res *http.Response) {
	r.Response = res
}

func (r *httpResponse) StatusCode() int {
	return r.Response.StatusCode
}

func (r *httpResponse) Header() http.Header {
	return r.Response.Header
}

func (r *httpResponse) Body() io.ReadCloser {
	return r.Response.Body
}

func (r *httpResponse) Trailer() http.Header {
	return r.Response.Trailer
}
