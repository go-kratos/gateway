package endpoint

import (
	"io"
	"net/http"
)

type httpRequest struct {
	*http.Request
}

// NewRequest new an HTTP request.
func NewRequest(req *http.Request) Request {
	return &httpRequest{req}
}

func (r *httpRequest) Path() string {
	return r.Request.RequestURI
}
func (r *httpRequest) Method() string {
	return r.Request.Method
}
func (r *httpRequest) Header() http.Header {
	return r.Request.Header
}
func (r *httpRequest) Body() io.ReadCloser {
	return r.Request.Body
}
