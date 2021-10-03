package mux

import (
	"net/http"
	"strings"

	"github.com/go-kratos/gateway/endpoint"
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

func (r *muxRouter) Handle(pattern string, method string, endpoint endpoint.Endpoint) {
	next := r.Router.NewRoute().Handler(endpoint)
	if strings.HasSuffix(pattern, "*") {
		// /api/echo/*
		next = next.PathPrefix(strings.TrimRight(pattern, "*"))
	} else {
		// /api/echo/hello
		// /api/echo/[a-z]+
		// /api/echo/{name}
		next = next.Path(pattern)
	}
	if method != "" {
		next = next.Methods(method)
	}
}
