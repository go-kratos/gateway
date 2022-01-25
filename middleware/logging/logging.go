package logging

import (
	"context"
	"net/http"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/logging/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	LOG = log.NewHelper(log.With(log.GetLogger(), "source", "accesslog"))
)

func init() {
	middleware.Register("logging", Middleware)
}

// Middleware is a logging middleware.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Logging{}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (reply *http.Response, err error) {
			reply, err = handler(ctx, req)
			startTime := time.Now()
			level := log.LevelInfo
			code := http.StatusBadGateway
			if err != nil {
				level = log.LevelError
			} else {
				code = reply.StatusCode
			}
			LOG.WithContext(ctx).Log(level,
				"method", req.Method,
				"scheme", req.URL.Scheme,
				"host", req.URL.Host,
				"path", req.URL.Path,
				"query", req.URL.RawQuery,
				"code", code,
				"latency", time.Since(startTime).Seconds(),
			)
			return reply, err
		}
	}, nil
}
