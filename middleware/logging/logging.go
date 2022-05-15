package logging

import (
	"net/http"
	"strings"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/logging/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	// LOG is access log
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
	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (reply *http.Response, err error) {
			startTime := time.Now()
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
			LOG.WithContext(ctx).Log(level,
				"host", req.Host,
				"method", req.Method,
				"scheme", req.URL.Scheme,
				"path", req.URL.Path,
				"query", req.URL.RawQuery,
				"code", code,
				"error", errMsg,
				"latency", time.Since(startTime).Seconds(),
				"backend", strings.Join(nodes, ","),
			)
			return reply, err
		})
	}, nil
}
