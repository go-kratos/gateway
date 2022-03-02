package circuitbreaker

import (
	"context"
	"net/http"

	"github.com/go-kratos/aegis/circuitbreaker/sre"
	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/circuitbreaker/v1"
	"github.com/go-kratos/gateway/client"
	"github.com/go-kratos/gateway/middleware"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func Init(in middleware.Factory) {
	middleware.Register("circuitbreaker", in)
}

func New(factory client.Factory) middleware.Factory {
	return middleware.Factory(func(c *config.Middleware) (middleware.Middleware, error) {
		options := &v1.CircuitBreaker{}
		if c.Options != nil {
			if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
				return nil, err
			}
		}

		succRatio := options.GetSuccessRatio()
		breaker := sre.NewBreaker(
			sre.WithBucket(int(succRatio.Bucket)),
			sre.WithRequest(succRatio.Request),
			sre.WithSuccess(succRatio.Success),
			sre.WithWindow(succRatio.Window.AsDuration()),
		)

		client, err := factory(options.GetBackupService().GetEndpoint())
		if err != nil {
			panic(err)
		}
		return func(handler middleware.Handler) middleware.Handler {
			return func(ctx context.Context, req *http.Request) (*http.Response, error) {
				if err := breaker.Allow(); err != nil {
					// rejected
					// NOTE: when client reject requets locally,
					// continue add counter let the drop ratio higher.
					breaker.MarkFailed()
					resp, err := client.Do(ctx, req)
					if err != nil {
						return nil, err
					}
					return resp, nil
				}
				resp, err := handler(ctx, req)
				if err != nil {
					breaker.MarkFailed()
					return nil, err
				}
				breaker.MarkSuccess()
				return resp, nil
			}
		}, nil
	})
}
