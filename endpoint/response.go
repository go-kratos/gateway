package endpoint

import (
	"io"
	"net/http"
)

type httpResponse struct {
	*http.Response
}

// NewResponse new an HTTP response.
func NewResponse(res *http.Response) Response {
	return &httpResponse{res}
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
