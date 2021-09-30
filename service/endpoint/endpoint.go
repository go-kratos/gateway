package endpoint

import (
	"context"
	"io"
	"net/http"

	"github.com/go-kratos/gateway/api"
	"github.com/go-kratos/gateway/service/backend"
	"github.com/go-kratos/gateway/service/middleware"

	km "github.com/go-kratos/kratos/v2/middleware"
)

type Endpoint struct {
	*api.Endpoint
}

func (e *Endpoint) Build() func(w http.ResponseWriter, r *http.Request) {
	b := backend.Backend{Backend: e.Backend}
	client := b.Build()
	h := func(ctx context.Context, req interface{}) (interface{}, error) {
		r := req.(*http.Request)
		return client.Do(r)
	}
	var middlewares []km.Middleware
	for _, m := range e.Middleware {
		mm := middleware.Middleware{Middleware: m}
		middlewares = append(middlewares, mm.Build())
	}
	handler := km.Chain(middlewares...)(h)
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := handler(r.Context(), r)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		resp := reply.(*http.Response)
		defer resp.Body.Close()
		header := w.Header()
		for k, v := range resp.Header {
			header[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		_, err = io.Copy(w, resp.Body)
	}
}
