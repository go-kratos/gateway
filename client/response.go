package client

import (
	"io"
	"net/http"
	"strconv"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
)

type Response struct {
	*http.Response
	protocol config.Protocol
}

func (r *Response) HTTPStatus() int {
	return r.Response.StatusCode
}

func (r *Response) GRPCStatus() uint32 {
	if status := r.Response.Header.Get("Grpc-Status"); status != "" {
		code, err := strconv.ParseUint(status, 10, 32)
		if err != nil {
			code = 13
		}
		return uint32(code)
	}
	return 0
}

func (r *Response) Protocol() config.Protocol {
	return r.protocol
}

func (r *Response) Header() http.Header {
	return r.Response.Header
}

func (r *Response) Body() io.ReadCloser {
	return r.Response.Body
}

func (r *Response) Trailer() http.Header {
	return r.Response.Trailer
}
