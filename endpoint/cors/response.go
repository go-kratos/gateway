package cors

import (
	"io"
	"net/http"
)

type response struct {
	statusCode int
	header     http.Header
}

func (r *response) Header() http.Header {
	return r.header
}

func (r *response) Trailer() http.Header {
	return nil
}

func (r *response) StatusCode() int {
	if r.statusCode == 0 {
		return 200
	}
	return r.statusCode
}

func (r *response) Body() io.ReadCloser {
	return nil
}

func newResponse(statusCode int, headers ...http.Header) *response {
	var header http.Header
	if len(headers) == 0 {
		header = make(http.Header)
	}
	return &response{statusCode: statusCode, header: header}
}
