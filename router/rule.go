package router

import (
	"strings"

	"github.com/gorilla/mux"
)

func (rule *RouteRule) Build() *mux.Router {
	router := mux.NewRouter()
	for _, ser := range rule.Service {
		var subs []*mux.Router
		for _, host := range ser.Host {
			subs = append(subs, router.Host(host).Subrouter())
		}
		// undefined behavior
		if len(subs) == 0 {
			subs = append(subs, router)
		}
		for _, e := range ser.Endpoint {
			for _, sub := range subs {
				if strings.HasSuffix(e.Path, "*") {
					sub.PathPrefix(strings.TrimRight(e.Path, "*")).HandlerFunc(e.Backend.Build())
				} else {
					sub.Path(e.Path).HandlerFunc(e.Backend.Build())
				}
			}
		}
	}
	return router
}
