package cors

import (
	"net/http"
)

func newResponse(statusCode int, header http.Header) *http.Response {
	return &http.Response{StatusCode: statusCode, Header: header}
}
