package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

type loggingKey struct{}

// NewLoggingContext returns a new Context that carries value.
func NewLoggingContext(ctx context.Context, logger log.Logger) context.Context {
	return context.WithValue(ctx, loggingKey{}, logger)
}

// FromLoggingContext returns the Transport value stored in ctx, if any.
func FromLoggingContext(ctx context.Context) (logger log.Logger, ok bool) {
	logger, ok = ctx.Value(loggingKey{}).(log.Logger)
	return
}
