package router

import "net/http"

// Router is a gateway router.
type Router interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
	Handle(pattern, method string, handler http.Handler)
}
