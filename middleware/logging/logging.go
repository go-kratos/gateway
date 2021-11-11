package logging

import (
	"context"
	"net/http"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/logging/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/log"
)

func init() {
	middleware.Register("logging", Middleware)
}

// Middleware is a logging middleware.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Logging{}
	if err := c.Options.UnmarshalTo(options); err != nil {
		return nil, err
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (reply *http.Response, err error) {
			reply, err = handler(ctx, req)
			logger, ok := middleware.FromLoggingContext(ctx)
			if ok {
				startTime := time.Now()
				level := log.LevelInfo
				code := http.StatusBadGateway
				if err != nil {
					level = log.LevelError
				} else {
					code = reply.StatusCode
				}
				_ = log.WithContext(ctx, logger).Log(level,
					"method", req.Method,
					"scheme", req.URL.Scheme,
					"host", req.URL.Host,
					"path", req.URL.Path,
					"query", req.URL.RawQuery,
					"code", code,
					"latency", time.Since(startTime).Seconds(),
				)
			}
			return reply, err
		}
	}, nil
}
