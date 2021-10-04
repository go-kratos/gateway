package proxy

import (
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/router"
	"github.com/go-kratos/gateway/router/mux"
)

// Proxy is a gateway proxy.
type Proxy struct {
	router  router.Router
	clients map[string]client.Client
}

// New new a gateway proxy.
func New() (*Proxy, error) {
	return &Proxy{router: mux.NewRouter()}, nil
}

// Update updates service endpoint.
func (p *Proxy) Update(services []*config.Service) error {
	router := mux.NewRouter()
	for _, s := range services {
		for _, e := range s.Endpoints {
			router.Handle(e.Path, e.Method, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				// TODO forword request
			}))
		}
	}
	p.router = router
	return nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.router.ServeHTTP(w, r)
}
