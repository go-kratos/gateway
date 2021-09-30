package service

import (
	"strings"

	"github.com/go-kratos/gateway/api"
	"github.com/go-kratos/gateway/service/endpoint"

	"github.com/gorilla/mux"
)

type Rule struct {
	*api.Rule
}

func (r *Rule) Build() *mux.Router {
	router := mux.NewRouter()
	for _, ser := range r.Service {
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
				var route *mux.Route
				if strings.HasSuffix(e.Path, "*") {
					route = sub.PathPrefix(strings.TrimRight(e.Path, "*"))
				} else {
					route = sub.Path(e.Path)
				}
				end := &endpoint.Endpoint{Endpoint: e}
				route.HandlerFunc(end.Build())
			}
		}
	}
	return router
}
