package router

import (
	"net/http"
)

// Router is a gateway router.
type Router interface {
	http.Handler
	Handle(pattern string, methods []string, handler http.Handler) error
}
