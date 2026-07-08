package exact

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

var _ router.Router = (*exactRouter)(nil)

type exactEntry struct {
	method  string
	host    string
	handler http.Handler
}

const anyMethod = ""
const anyHost = ""

type exactRouter struct {
	*mux.Router
	exactHandlers map[string][]exactEntry

	wg        *sync.WaitGroup
	allCloser []io.Closer
}

// Option is exact router option.
type Option func(*exactRouter)

// NewRouter creates a router that resolves exact routes (literal path, single
// method, optional host) in O(1) before falling through to gorilla/mux's
// ordered route matcher. Non-exact routes (prefix, wildcard, template, regexp)
// are handled by mux as usual.
func NewRouter(notFoundHandler, methodNotAllowedHandler http.Handler, opts ...Option) router.Router {
	r := &exactRouter{
		Router:        mux.NewRouter().StrictSlash(EnableStrictSlash),
		exactHandlers: make(map[string][]exactEntry),
		wg:            &sync.WaitGroup{},
	}
	for _, opt := range opts {
		opt(r)
	}
	r.Router.Handle("/metrics", protectedHandler(promhttp.Handler()))
	r.Router.NotFoundHandler = notFoundHandler
	r.Router.MethodNotAllowedHandler = methodNotAllowedHandler
	return r
}

func protectedHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-For") != "" {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
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

// ServeHTTP implements http.Handler.
func (r *exactRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.wg.Add(1)
	defer r.wg.Done()
	req.URL.Path = cleanPath(req.URL.Path)
	if handler, ok := r.matchExact(req); ok {
		handler.ServeHTTP(w, req)
		return
	}
	r.Router.ServeHTTP(w, req)
}

// Handle implements router.Router.
func (r *exactRouter) Handle(pattern, method, host string, handler http.Handler, closer io.Closer) error {
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
	r.registerExact(next, handler)
	r.allCloser = append(r.allCloser, closer)
	return nil
}

// SyncClose implements router.Router.
func (r *exactRouter) SyncClose(ctx context.Context) error {
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

// --- exact match helpers ---

func (r *exactRouter) registerExact(route *mux.Route, handler http.Handler) {
	pathTemplate, err := route.GetPathTemplate()
	if err != nil || !isExactPathTemplate(pathTemplate) {
		return
	}
	pathRegexp, err := route.GetPathRegexp()
	if err != nil || !strings.HasSuffix(pathRegexp, "$") {
		return
	}
	methods, err := route.GetMethods()
	if err != nil {
		methods = []string{anyMethod}
	}
	host, err := route.GetHostTemplate()
	if err != nil {
		host = anyHost
	}
	if !isExactHostTemplate(host) {
		return
	}
	p := cleanPath(pathTemplate)
	for _, method := range methods {
		r.exactHandlers[p] = append(r.exactHandlers[p], exactEntry{
			method:  method,
			host:    host,
			handler: handler,
		})
	}
}

func (r *exactRouter) matchExact(req *http.Request) (http.Handler, bool) {
	path := req.URL.Path
	host := getHost(req)
	for _, entry := range r.exactHandlers[path] {
		if entry.method != anyMethod && entry.method != req.Method {
			continue
		}
		if entry.host != anyHost && entry.host != host {
			continue
		}
		return entry.handler, true
	}
	return nil, false
}

func getHost(r *http.Request) string {
	if r.URL.IsAbs() {
		return r.URL.Host
	}
	return r.Host
}

func isExactPathTemplate(pattern string) bool {
	return !strings.ContainsAny(pattern, "*{}[]()")
}

func isExactHostTemplate(host string) bool {
	return !strings.ContainsAny(host, "{}[]()")
}
