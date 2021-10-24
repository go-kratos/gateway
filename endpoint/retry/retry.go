package retry

import (
	"context"
	"errors"
	"net"
	"syscall"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/retry/v1"
	"github.com/go-kratos/gateway/endpoint"
	"github.com/go-kratos/kratos/v2/selector"
)

// Name is the middleware name.
const Name = "retry"

func retryable(c *v1.Retry, reply endpoint.Response, err error) bool {
	if err != nil {
		// connect error
		var oe *net.OpError
		if !errors.As(err, &oe) {
			return false
		}
		if oe.Op != "dial" {
			return false
		}

		if oe.Timeout() {
			return true
		}

		if errors.Is(oe, syscall.ECONNREFUSED) || errors.Is(oe, syscall.ETIMEDOUT) {
			return true
		}
		return false
	}

	// response head
	h := reply.Header()
	for _, v := range c.RetriableHeaders {
		if h.Get(v) != "" {
			return true
		}
	}

	// http code
	code := reply.StatusCode()
	for _, v := range c.RetriableStatusCodes {
		if code == int(v) {
			return true
		}
	}

	// grpc status
	grpcst := reply.Trailer().Get("grpc-status")
	if len(grpcst) > 0 {
		tmp := grpcst[0] - '0'
		if len(grpcst) > 1 {
			tmp = tmp*10 + (grpcst[1] - '0')
		}

		for _, v := range c.RetriableGrpcStatus {
			if tmp == byte(v) {
				return true
			}
		}
	}

	return false
}

// Middleware .
func Middleware(c *config.Middleware) (endpoint.Middleware, error) {
	options := &v1.Retry{}
	if err := c.Options.UnmarshalTo(options); err != nil {
		return nil, err
	}

	var maxAttempt uint32
	if tmp := options.NumRetries; tmp != nil {
		maxAttempt = tmp.Value + 1
	} else {
		maxAttempt = 2
	}

	return func(handler endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req endpoint.Request) (reply endpoint.Response, err error) {
			opts, _ := endpoint.FromContext(ctx)

			for i := uint32(0); i < maxAttempt; i++ {
				reply, err = handler(ctx, req)
				if i == maxAttempt-1 || !retryable(options, reply, err) {
					break
				}

				// TODO: log

				// add retry filter
				if options.PreviousHosts && i == 0 {
					opts.Filters = append(opts.Filters, func(ctx context.Context, ns []selector.Node) []selector.Node {
						if len(ns) <= 1 {
							return ns
						}

						// copy or inplace ?
						ret := make([]selector.Node, 0, len(ns))
						for _, n := range ns {
							var ignore bool
							for _, u := range opts.UsedNodes {
								if n == u {
									ignore = true
									break
								}
							}
							if !ignore {
								ret = append(ret, n)
							}
						}

						if len(ret) == 0 {
							return ns
						}

						return ret
					})
				}
			}
			return
		}
	}, nil
}
