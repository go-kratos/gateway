package router

import (
	"net/http"
)

// Router is a gateway router.
type Router interface {
	http.Handler
	Handle(pattern, method, host string, handler http.Handler) error
}
