package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/transport/http/status"
)

func writeError(w http.ResponseWriter, r *http.Request, e *config.Endpoint, err error, observer Observer) {
	var statusCode int
	switch {
	case errors.Is(err, context.Canceled),
		err.Error() == "client disconnected":
		statusCode = 499
	case errors.Is(err, context.DeadlineExceeded):
		statusCode = 504
	default:
		log.Error(fmt.Sprintf("Failed to handle request: %s: %+v", r.URL.String(), err))
		statusCode = 502
	}
	observer.HandleRequest(r, w.Header(), statusCode, err)
	if e.Protocol == config.Protocol_GRPC {
		// see https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
		code := strconv.Itoa(int(status.ToGRPCCode(statusCode)))
		w.Header().Set("Content-Type", "application/grpc")
		w.Header().Set("Grpc-Status", code)
		w.Header().Set("Grpc-Message", err.Error())
		statusCode = 200
	}
	w.WriteHeader(statusCode)
}

// notFoundHandler replies to the request with an HTTP 404 not found error.
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	code := http.StatusNotFound
	message := "404 page not found"
	http.Error(w, message, code)
	log.ErrorContext(r.Context(), message,
		"source", "accesslog",
		"host", r.Host,
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"user_agent", r.Header.Get("User-Agent"),
		"code", code,
		"error", message,
	)
	MetricRequestsTotal.WithLabelValues("HTTP", r.Method, "/404", strconv.Itoa(code), "", "").Inc()
}

func methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	code := http.StatusMethodNotAllowed
	message := http.StatusText(code)
	http.Error(w, message, code)
	log.ErrorContext(r.Context(), message,
		"source", "accesslog",
		"host", r.Host,
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"user_agent", r.Header.Get("User-Agent"),
		"code", code,
		"error", message,
	)
	MetricRequestsTotal.WithLabelValues("HTTP", r.Method, "/405", strconv.Itoa(code), "", "").Inc()
}
