package router

import (
	"net/http"
	"sync/atomic"

	"github.com/gorilla/mux"
)

type RuleSource interface {
	Load() (*RouteRule, error)
	Watch(func(rule *RouteRule)) error
}

type Router struct {
	router atomic.Value
	source RuleSource
	hosts  map[string]atomic.Value
}

func New(ruleSource RuleSource) (*Router, error) {
	r := &Router{
		source: ruleSource,
	}
	rule, err := r.source.Load()
	if err != nil {
		return nil, err
	}
	r.router.Store(rule.Build())

	err = r.source.Watch(func(rule *RouteRule) {
		r.router.Store(rule.Build())
	})

	return r, err
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	router := r.router.Load().(*mux.Router)
	router.ServeHTTP(w, req)
}
