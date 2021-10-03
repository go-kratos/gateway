package router

import (
	"net/http"

	"github.com/go-kratos/gateway/endpoint"
)

// Router is a gateway router.
type Router interface {
	http.Handler
	Handle(pattern, method string, endpoint endpoint.Endpoint)
}
