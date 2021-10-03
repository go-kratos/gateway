package proxy

import (
	"net/http"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/router"
)

// Proxy is a gateway proxy.
type Proxy struct {
	r router.Router
}

// New new a gateway proxy.
func New(c []*config.Service) (*Proxy, error) {
	// TODO init endpoint & middleware
	return &Proxy{}, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.r.ServeHTTP(w, r)
}
