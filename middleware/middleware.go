package middleware

import (
	"context"
	"io"
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
)

// type Middleware func(http.Handler) http.Handler

// Handler defines the handler invoked by Middleware.
type Handler func(ctx context.Context, req *http.Request) (Response, error)

// Middleware is HTTP/gRPC transport middleware.
type Middleware func(Handler) Handler

/*type Request interface {
	Method() string
	Protocol() config.Protocol
	Endpoint() *url.URL
	Header() http.Header
	Body() io.ReadCloser
	Host() string
}*/

type Response interface {
	HTTPStatus() int
	GRPCStatus() uint32
	Protocol() config.Protocol
	Header() http.Header
	Body() io.ReadCloser
	Trailer() http.Header
}
