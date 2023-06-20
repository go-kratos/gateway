package router

import (
	"context"
	"net/http"
)

// Router is a gateway router.
type Router interface {
	http.Handler
	Handle(pattern, method, host string, handler http.Handler, closeFn func() error) error
	SyncClose(ctx context.Context) error
}
