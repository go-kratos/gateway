package mux

import (
	"net/http"
	"strings"

	"github.com/go-kratos/gateway/router"
	"github.com/gorilla/mux"
)

var _ = new(router.Router)

type muxRouter struct {
	*mux.Router
}

// NewRouter new a mux router.
func NewRouter() router.Router {
	return &muxRouter{
		Router: mux.NewRouter().StrictSlash(true),
	}
}

func (r *muxRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Router.ServeHTTP(w, req)
}

func (r *muxRouter) Handle(pattern string, methods []string, handler http.Handler) error {
	next := r.Router.NewRoute().Handler(handler)
	if strings.HasSuffix(pattern, "*") {
		// /api/echo/*
		next = next.PathPrefix(strings.TrimRight(pattern, "*"))
	} else {
		// /api/echo/hello
		// /api/echo/[a-z]+
		// /api/echo/{name}
		next = next.Path(pattern)
	}
	if len(methods) > 0 {
		next = next.Methods(methods...)
	}
	return next.GetError()
}
