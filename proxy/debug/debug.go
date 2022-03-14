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

var LOG = log.NewHelper(log.With(log.GetLogger(), "source", "debug"))

type Debuggable interface {
	DebugHandler() http.Handler
}

type DebugService struct {
	handlers map[string]http.HandlerFunc
	mux      *mux.Router
}

func New() *DebugService {
	return &DebugService{
		handlers: map[string]http.HandlerFunc{
			"/debug/ping":          func(rw http.ResponseWriter, r *http.Request) {},
			"/debug/pprof/":        pprof.Index,
			"/debug/pprof/cmdline": pprof.Cmdline,
			"/debug/pprof/profile": pprof.Profile,
			"/debug/pprof/symbol":  pprof.Symbol,
			"/debug/pprof/trace":   pprof.Trace,
		},
		mux: mux.NewRouter(),
	}
}

func (d *DebugService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for path, handler := range d.handlers {
		if path == req.URL.Path {
			handler(w, req)
			return
		}
	}
	d.mux.ServeHTTP(w, req)
}

func (d *DebugService) Register(name string, component interface{}) {
	debuggable, ok := component.(Debuggable)
	if !ok {
		LOG.Warnf("component %s is not debuggable", name)
		return
	}
	path := path.Join(_debugPrefix, name)
	LOG.Infof("register debug: %s", path)
	d.mux.PathPrefix(path).Handler(debuggable.DebugHandler())
}

func MashupWithDebugHandler(debug *DebugService, origin http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, _debugPrefix) {
			debug.ServeHTTP(w, req)
			return
		}
		origin.ServeHTTP(w, req)
	})
}
