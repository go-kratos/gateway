package proxy

import "net/http"

type Request struct {
	*http.Request
}
