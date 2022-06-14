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

//type ecodeResponseBody struct {
//	Code    int32  `json:"code"`
//	Message string `json:"message"`
//	Reason  string `json:"reason"`
//}
//
//func newEcodeResponseBody(err errors.Error) *ecodeResponseBody {
//	return &ecodeResponseBody{
//		Code:    err.Code,
//		Message: err.Message,
//		Reason:  err.Reason,
//	}
//}
//func WriteJSON(in io.Writer) error {
//	return json.NewEncoder(in).Encode()
//}

func MakeResonse(err *errors.Error) *http.Response {
	return &http.Response{
		StatusCode: int(err.Code),
		Status:     http.StatusText(int(err.Code)),
		Body:       _nopBody,
	}
}
