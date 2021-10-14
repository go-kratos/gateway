package tracing

import (
	"context"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/tracing/v1"
	"github.com/go-kratos/gateway/endpoint"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Name is the middleware name
const (
	Name       = "opentelemetry"
	TracerName = "gateway"
)

func Middleware(cfg *config.Middleware) (endpoint.Middleware, error) {
	options := &v1.Tracing{}
	if err := cfg.Options.UnmarshalTo(options); err != nil {
		return nil, errors.WithStack(err)
	}

	var sampler sdktrace.Sampler
	if options.SampleRatio != 0 {
		sampler = sdktrace.TraceIDRatioBased(float64(options.SampleRatio))
	} else {
		sampler = sdktrace.AlwaysSample()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)
	tracer := otel.Tracer(TracerName)

	return func(handler endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req endpoint.Request) (reply endpoint.Response, err error) {
			nCtx, span := tracer.Start(ctx, req.Path())
			defer span.End()
			return handler(nCtx, req)
		}
	}, nil
}
