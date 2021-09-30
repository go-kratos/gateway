package service

import (
	"net/http"
	"sync/atomic"

	"github.com/go-kratos/gateway/api"

	"github.com/gorilla/mux"
)

type RuleSource interface {
	Load() (*api.Rule, error)
	Watch(func(rule *api.Rule)) error
}

type Service struct {
	router atomic.Value
	source RuleSource
}

func New(source RuleSource) (*Service, error) {
	router := &Service{
		source: source,
	}
	rule, err := router.source.Load()
	if err != nil {
		return nil, err
	}
	r := &Rule{rule}
	router.router.Store(r.Build())

	err = router.source.Watch(func(rule *api.Rule) {
		r := &Rule{rule}
		router.router.Store(r.Build())
	})

	return router, err
}

func (r *Service) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	router := r.router.Load().(*mux.Router)
	router.ServeHTTP(w, req)
}
