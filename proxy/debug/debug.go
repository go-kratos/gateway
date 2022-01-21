package debug

import (
	"net/http"
	"path"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/mux"
)

const (
	_debugPrefix = "/_/debug"
)

var LOG = log.NewHelper(log.With(log.GetLogger(), "source", "debug"))

type DebugService struct {
	mux *mux.Router
}

func New() *DebugService {
	return &DebugService{mux: mux.NewRouter()}
}

func (d *DebugService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	d.mux.ServeHTTP(w, req)
}

type Debuggable interface {
	DebugHandler() http.Handler
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

func (d *DebugService) Register(name string, component interface{}) {
	debuggable, ok := component.(Debuggable)
	if !ok {
		LOG.Warnf("component %s is not debuggable", name)
		return
	}
	d.mux.PathPrefix(path.Join(_debugPrefix, name)).Handler(debuggable.DebugHandler())
}
