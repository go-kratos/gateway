package proxy

import (
	"context"
	"net/http"
	"time"
)

// AttemptTimeoutContext is a function type that prepares a context with timeout for an HTTP request.
type AttemptTimeoutContext func(ctx context.Context, req *http.Request, timeout time.Duration) (context.Context, context.CancelFunc)

type interceptors struct {
	prepareAttemptTimeoutContext AttemptTimeoutContext
}

func (i *interceptors) SetPrepareAttemptTimeoutContext(f AttemptTimeoutContext) {
	if f != nil {
		i.prepareAttemptTimeoutContext = f
	}
}
