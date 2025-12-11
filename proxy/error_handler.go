package proxy

import (
	"net/http"
	"strconv"

	"github.com/go-kratos/kratos/v2/log"
)

// notFoundHandler replies to the request with an HTTP 404 not found error.
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	code := http.StatusNotFound
	message := "404 page not found"
	http.Error(w, message, code)
	log.Context(r.Context()).Errorw(
		"source", "accesslog",
		"host", r.Host,
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"user_agent", r.Header.Get("User-Agent"),
		"code", code,
		"error", message,
	)
	metricRequestsTotal.WithLabelValues("HTTP", r.Method, "/404", strconv.Itoa(code), "", "").Inc()
}

func methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	code := http.StatusMethodNotAllowed
	message := http.StatusText(code)
	http.Error(w, message, code)
	log.Context(r.Context()).Errorw(
		"source", "accesslog",
		"host", r.Host,
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"user_agent", r.Header.Get("User-Agent"),
		"code", code,
		"error", message,
	)
	metricRequestsTotal.WithLabelValues("HTTP", r.Method, "/405", strconv.Itoa(code), "", "").Inc()
}
