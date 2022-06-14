package bbr

import (
	"github.com/go-kratos/aegis/ratelimit"
	"github.com/go-kratos/gateway/errors"
	"net/http"

	"github.com/go-kratos/aegis/ratelimit/bbr"
	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

func init() {
	middleware.Register("bbr", Middleware)
}

func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	limiter := bbr.NewLimiter() //use default settings
	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			done, err := limiter.Allow()
			if err != nil {
				return errors.MakeResponse(errors.ErrLimitExceed), errors.ErrLimitExceed
			}
			resp, err := next.RoundTrip(req)
			done(ratelimit.DoneInfo{Err: err})
			return resp, err
		})
	}, nil
}
