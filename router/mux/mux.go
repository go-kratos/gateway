package mux

import (
	"net/http"
	"path"
	"strings"

	"github.com/go-kratos/gateway/router"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var _ = new(router.Router)

type muxRouter struct {
	*mux.Router
}

// NewRouter new a mux router.
func NewRouter(notFoundHandler, methodNotAllowedHandler http.Handler) router.Router {
	r := &muxRouter{
		Router: mux.NewRouter().StrictSlash(true),
	}
	r.Router.Handle("/metrics", promhttp.Handler())
	r.Router.NotFoundHandler = notFoundHandler
	r.Router.MethodNotAllowedHandler = methodNotAllowedHandler
	return r
}

func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}

	return np
}

func (r *muxRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req.URL.Path = cleanPath(req.URL.Path)
	r.Router.ServeHTTP(w, req)
}

func (r *muxRouter) Handle(pattern, method string, handler http.Handler) error {
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
	if method != "" && method != "*" {
		next = next.Methods(method, http.MethodOptions)
	}
	return next.GetError()
}

type RouterInspect struct {
	PathTemplate     string   `json:"path_template"`
	PathRegexp       string   `json:"path_regexp"`
	QueriesTemplates []string `json:"queries_templates"`
	QueriesRegexps   []string `json:"queries_regexps"`
	Methods          []string `json:"methods"`
}

func InspectMuxRouter(in interface{}) []*RouterInspect {
	r, ok := in.(*muxRouter)
	if !ok {
		return nil
	}
	var out []*RouterInspect
	_ = r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, _ := route.GetPathTemplate()
		pathRegexp, _ := route.GetPathRegexp()
		queriesTemplates, _ := route.GetQueriesTemplates()
		queriesRegexps, _ := route.GetQueriesRegexp()
		methods, _ := route.GetMethods()
		out = append(out, &RouterInspect{
			PathTemplate:     pathTemplate,
			PathRegexp:       pathRegexp,
			QueriesTemplates: queriesTemplates,
			QueriesRegexps:   queriesRegexps,
			Methods:          methods,
		})
		return nil
	})
	return out
}
