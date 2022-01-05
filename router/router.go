package router

import (
	"net/http"
)

// Router is a gateway router.
type Router interface {
	http.Handler
	Handle(pattern, method string, handler http.Handler) error
}
