package router

import (
	"context"
	"io"
	"net/http"
)

// Router is a gateway router.
type Router interface {
	http.Handler
	Handle(pattern, method, host string, handler http.Handler, closer io.Closer) error
	SyncClose(ctx context.Context) error
}
