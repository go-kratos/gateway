package router

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/wrr"
)

var _ selector.Node = &node{}

type node struct {
	address  string
	name     string
	weight   *int64
	version  string
	metadata map[string]string
}

func (n *node) Address() string {
	return n.address
}

// ServiceName is service name
func (n *node) ServiceName() string {
	return n.name
}

// InitialWeight is the initial value of scheduling weight
// if not set return nil
func (n *node) InitialWeight() *int64 {
	return n.weight
}

// Version is service node version
func (n *node) Version() string {
	return n.version
}

// Metadata is the kv pair metadata associated with the service instance.
// version,namespace,region,protocol etc..
func (n *node) Metadata() map[string]string {
	return n.metadata
}

func (backend *Backend) Build() func(w http.ResponseWriter, r *http.Request) {
	s := wrr.New()
	var nodes []selector.Node
	for _, target := range backend.Target {
		nodes = append(nodes, &node{address: target})
	}
	s.Apply(nodes)
	var client http.Client
	return func(w http.ResponseWriter, r *http.Request) {
		selected, done, err := s.Select(r.Context())
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		defer done(r.Context(), selector.DoneInfo{Err: err})
		req, err := http.NewRequest(r.Method, fmt.Sprintf("%s://%s%s", backend.Scheme, selected.Address(), r.URL.RawPath), r.Body)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		resp, err := client.Do(req)
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
