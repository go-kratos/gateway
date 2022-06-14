package errors

import (
	"bytes"
	"github.com/go-kratos/kratos/v2/errors"
	"io"
	"net/http"
)

var (
	ErrLimitExceed = errors.New(429, "RATELIMIT", "service unavailable due to rate limit exceeded")
	_nopBody       = io.NopCloser(&bytes.Buffer{})
)

func MakeResponse(err *errors.Error) *http.Response {
	return &http.Response{
		StatusCode: int(err.Code),
		Status:     http.StatusText(int(err.Code)),
		Body:       _nopBody,
	}
}
