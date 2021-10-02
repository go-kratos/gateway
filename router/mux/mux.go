package mux

import (
	"net/http"

	"github.com/go-kratos/gateway/router"
	"github.com/gorilla/mux"
)

type muxRouter struct {
	*mux.Router
}

// NewRouter new a mux router.
func NewRouter() router.Router {
	return &muxRouter{Router: mux.NewRouter()}
}

func (r *muxRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Router.ServeHTTP(w, req)
}

func (r *muxRouter) Handle(pattern, method string, handler http.Handler) {
	r.Router.Handle(pattern, handler).Methods(method)
}
