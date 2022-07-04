package logging

import (
	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/nanotime"
	"github.com/go-kratos/kratos/v2/log"
	"net/http"
	"strings"
)

func init() {
	middleware.Register("logging", Middleware)
}

// Middleware is a logging middleware.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (reply *http.Response, err error) {
			startTime := nanotime.RuntimeNanotime()
			reply, err = next.RoundTrip(req)
			level := log.LevelInfo
			code := http.StatusBadGateway
			errMsg := ""
			if err != nil {
				level = log.LevelError
				errMsg = err.Error()
			} else {
				code = reply.StatusCode
			}
			ctx := req.Context()
			nodes, _ := middleware.RequestBackendsFromContext(ctx)
			log.Context(ctx).Log(level,
				"source", "accesslog",
				"host", req.Host,
				"method", req.Method,
				"scheme", req.URL.Scheme,
				"path", req.URL.Path,
				"query", req.URL.RawQuery,
				"code", code,
				"error", errMsg,
				"latency", nanotime.SinceSeconds(startTime),
				"backend", strings.Join(nodes, ","),
			)
			return reply, err
		})
	}, nil
}
