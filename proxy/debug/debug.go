package debug

import (
	"net/http"
	"net/http/pprof"
	"path"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/mux"
)

const (
	_debugPrefix = "/debug"
)

var globalService = &debugService{
	handlers: map[string]http.HandlerFunc{
		"/debug/ping":               func(rw http.ResponseWriter, r *http.Request) {},
		"/debug/pprof/":             pprof.Index,
		"/debug/pprof/cmdline":      pprof.Cmdline,
		"/debug/pprof/profile":      pprof.Profile,
		"/debug/pprof/symbol":       pprof.Symbol,
		"/debug/pprof/trace":        pprof.Trace,
		"/debug/pprof/allocs":       pprof.Handler("allocs").ServeHTTP,
		"/debug/pprof/block":        pprof.Handler("block").ServeHTTP,
		"/debug/pprof/goroutine":    pprof.Handler("goroutine").ServeHTTP,
		"/debug/pprof/heap":         pprof.Handler("heap").ServeHTTP,
		"/debug/pprof/mutex":        pprof.Handler("mutex").ServeHTTP,
		"/debug/pprof/threadcreate": pprof.Handler("threadcreate").ServeHTTP,
	},
	mux: mux.NewRouter(),
}

func Register(name string, debuggable Debuggable) {
	globalService.Register(name, debuggable)
}

func MashupWithDebugHandler(origin http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, _debugPrefix) {
			globalService.ServeHTTP(w, req)
			return
		}
		origin.ServeHTTP(w, req)
	})
}

type Debuggable interface {
	DebugHandler() http.Handler
}

type debugService struct {
	handlers map[string]http.HandlerFunc
	mux      *mux.Router
}

func (d *debugService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for path, handler := range d.handlers {
		if path == req.URL.Path {
			handler(w, req)
			return
		}
	}
	d.mux.ServeHTTP(w, req)
}

func (d *debugService) Register(name string, debuggable Debuggable) {
	path := path.Join(_debugPrefix, name)
	d.mux.PathPrefix(path).Handler(debuggable.DebugHandler())
	log.Infof("register debug: %s", path)
}
