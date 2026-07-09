package exact_mux

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type PreRouter interface {
	Register(*mux.Route, http.Handler)
	Match(*http.Request) (http.Handler, bool)
	Inspect() json.RawMessage
}

type nopRouter struct{}

func NewNopRouter() nopRouter                                { return nopRouter{} }
func (n nopRouter) Register(*mux.Route, http.Handler)        {}
func (n nopRouter) Match(*http.Request) (http.Handler, bool) { return nil, false }
func (n nopRouter) Inspect() json.RawMessage                 { return nil }

var _ PreRouter = (*nopRouter)(nil)

// exactRouter is a pre-router for gorilla/mux that resolves exact routes before
// the request enters mux's ordered route matcher. It is used only for routes
// that can be matched accurately by method, host, and path.
// Note: exactRouter is currently experimental. Use it at your own risk.
//
// Gorilla mux evaluates routes in registration order and stops at the first
// route that matches. That is correct for mux's general matcher model, but it
// means exact gateway routes can still be order-dependent when they share a path
// and differ only by host or method scope. A hostless route matches requests for
// every host, and a methodless route matches every method, so either one can
// shadow a more specific route if it was registered first.
//
// exactRouter preserves that mux behavior by grouping exact routes by path and
// keeping each path bucket in registration order. Matching scans only the bucket
// for the request path and returns the first entry whose method and host scopes
// match the request.
//
// Routes that are not exact, such as path-prefix, wildcard, template, or regexp
// routes, should remain handled by gorilla mux because their precedence still
// depends on mux's full matcher semantics.
type exactRouter struct {
	handlers map[string][]exactEntry
}

// NewExactRouter returns an experimental exact pre-router. Use it at your own
// risk.
func NewExactRouter() PreRouter {
	return &exactRouter{
		handlers: make(map[string][]exactEntry),
	}
}

type exactEntry struct {
	method  string
	host    string
	handler http.Handler
}

const anyMethod = ""
const anyHost = ""

func (e *exactRouter) Register(route *mux.Route, handler http.Handler) {
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
	path := cleanPath(pathTemplate)
	for _, method := range methods {
		e.handlers[path] = append(e.handlers[path], exactEntry{
			method:  method,
			host:    host,
			handler: handler,
		})
	}
}

func (e *exactRouter) Match(r *http.Request) (http.Handler, bool) {
	path := r.URL.Path
	host := getHost(r)
	for _, entry := range e.handlers[path] {
		if entry.method != anyMethod && entry.method != r.Method {
			continue
		}
		if entry.host != anyHost && entry.host != host {
			continue
		}
		return entry.handler, true
	}
	return nil, false
}

func (e *exactRouter) Inspect() json.RawMessage {
	type inspectEntry struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Host   string `json:"host"`
	}
	out := make([]inspectEntry, 0)
	for path, entries := range e.handlers {
		for _, entry := range entries {
			out = append(out, inspectEntry{
				Method: entry.method,
				Path:   path,
				Host:   entry.host,
			})
		}
	}
	data, _ := json.Marshal(out)
	return data
}

var _ PreRouter = (*exactRouter)(nil)

func getHost(r *http.Request) string {
	if r.URL.IsAbs() {
		return r.URL.Host
	}
	return r.Host
}

// Exact path templates are the subset that mux will compile to a literal,
// anchored regexp and that gateway did not register as a prefix route. The
// relevant mux path is:
//
//	Route.Path -> addRegexpMatcher -> newRouteRegexp -> braceIndices
//
// In newRouteRegexp, "{name}" and "{name:regexp}" are parsed as variables, and
// the regexp part may contain "[]()" or other regexp syntax. Those templates
// cannot be looked up by a raw method/host/path key. Gateway route patterns
// ending in "*" are registered with Route.PathPrefix instead of Route.Path, so
// "*" is also excluded for paths.
func isExactPathTemplate(pattern string) bool {
	return !strings.ContainsAny(pattern, "*{}[]()")
}

// Exact host templates follow the same mux parser path as paths:
//
//	Route.Host -> addRegexpMatcher -> newRouteRegexp -> braceIndices
//
// For hosts, mux treats "{name}" and "{name:regexp}" as variables and uses a
// host-specific default variable pattern. The custom regexp part can contain
// "[]()" or other regexp syntax, so those templates cannot be indexed as a raw
// host string. Unlike paths, hosts do not use gateway's trailing "*" prefix
// convention here, so "*" is not filtered.
func isExactHostTemplate(host string) bool {
	return !strings.ContainsAny(host, "{}[]()")
}
