package logging

import (
	"context"
	"net/http"
	"strings"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/logging/v1"
	"github.com/go-kratos/gateway/endpoint"
	"github.com/go-kratos/kratos/v2/log"
)

// Name is the middleware name.
const Name = "logging"

func init() {
	endpoint.Register(Name, Middleware)
}

func Middleware(c *config.Middleware) (endpoint.Middleware, error) {
	options := &v1.Logging{}
	if err := c.Options.UnmarshalTo(options); err != nil {
		return nil, err
	}
	logger, err := NewFileLogger(options.Path)
	if err != nil {
		return nil, err
	}
	return func(handler endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req *http.Request) (reply *http.Response, err error) {
			startTime := time.Now()
			level := log.LevelInfo
			reply, err = handler(ctx, req)
			if err != nil {
				level = log.LevelError
			}

			_ = log.WithContext(ctx, logger).Log(level,
				"method", req.Method,
				"scheme", req.URL.Scheme,
				"host", req.URL.Host,
				"path", req.URL.Path,
				"query", strings.Split(req.URL.RawQuery, "&"),
				"code", reply.StatusCode,
				"latency", time.Since(startTime).Seconds(),
			)
			return reply, err
		}
	}, nil
}
