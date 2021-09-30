package middleware

import (
	"context"

	"github.com/go-kratos/gateway/api"

	"github.com/go-kratos/kratos/v2/middleware"
)

type Middleware struct {
	*api.Middleware
}

func (m *Middleware) Build() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			// TODO: add impl
			return
		}
	}
}
