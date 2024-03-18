package mux

import (
	"context"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/go-kratos/gateway/router"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var EnableStrictSlash = parseBool(os.Getenv("ENABLE_STRICT_SLASH"), false)

func parseBool(in string, defV bool) bool {
	if in == "" {
		return defV
	}
	v, err := strconv.ParseBool(in)
	if err != nil {
		return defV
	}
	return v
}

var _ = new(router.Router)

type muxRouter struct {
	*mux.Router
	wg        *sync.WaitGroup
	allCloser []io.Closer
}

func ProtectedHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-For") != "" {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// NewRouter new a mux router.
func NewRouter(notFoundHandler, methodNotAllowedHandler http.Handler) router.Router {
	r := &muxRouter{
		Router: mux.NewRouter().StrictSlash(EnableStrictSlash),
		wg:     &sync.WaitGroup{},
	}
	r.Router.Handle("/metrics", ProtectedHandler(promhttp.Handler()))
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
	r.wg.Add(1)
	defer r.wg.Done()
	req.URL.Path = cleanPath(req.URL.Path)
	r.Router.ServeHTTP(w, req)
}

func (r *muxRouter) Handle(pattern, method, host string, handler http.Handler, closer io.Closer) error {
	next := r.Router.NewRoute().Handler(handler)
	if host != "" {
		next = next.Host(host)
	}
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
	if err := next.GetError(); err != nil {
		return err
	}
	r.allCloser = append(r.allCloser, closer)
	return nil
}

func (r *muxRouter) SyncClose(ctx context.Context) error {
	if timeout := waitTimeout(ctx, r.wg); timeout {
		log.Warnf("Time out to wait all requests complete, processing force close")
	}
	for _, closer := range r.allCloser {
		if err := closer.Close(); err != nil {
			log.Errorf("Failed to execute close function: %+v", err)
			continue
		}
	}
	return nil
}

func waitTimeout(ctx context.Context, wg *sync.WaitGroup) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-ctx.Done():
		return true // timed out
	}
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
