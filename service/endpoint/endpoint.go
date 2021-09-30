package endpoint

import (
	"io"
	"net/http"

	"github.com/go-kratos/gateway/api"
	"github.com/go-kratos/gateway/service/backend"
)

type Endpoint struct {
	*api.Endpoint
}

func (e *Endpoint) Build() func(w http.ResponseWriter, r *http.Request) {
	b := backend.Backend{e.Backend}
	client := b.Build()
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.Do(r)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		defer resp.Body.Close()
		header := w.Header()
		for k, v := range resp.Header {
			header[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		_, err = io.Copy(w, resp.Body)
	}
}
