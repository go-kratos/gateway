package otel

import (
	"context"
	"log"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/tracing/v1"
	"github.com/go-kratos/gateway/endpoint"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Name is the middleware name
const Name = "opentelemetry"

func Middleware(cfg *config.Middleware) (endpoint.Middleware, error) {
	options := &v1.Tracing{}
	if err := cfg.Options.UnmarshalTo(options); err != nil {
		return nil, errors.WithStack(err)
	}

	tracer := NewTracer(context.Background(), options)
	// propagator := propagation.NewCompositeTextMapPropagator()

	return func(handler endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req endpoint.Request) (reply endpoint.Response, err error) {
			ctx, span := tracer.Start(ctx, req.Path(), trace.WithSpanKind(trace.SpanKindClient))
			defer func() {
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
				} else {
					span.SetStatus(codes.Ok, "OK")
				}
				span.End()
			}()
			return handler(ctx, req)
		}
	}, nil
}

func NewTracer(ctx context.Context, options *v1.Tracing) trace.Tracer {
	var (
		serviceName = "gateway"
		timeout     = time.Duration(10 * time.Second)
	)

	var sampler sdktrace.Sampler
	if options.SampleRatio != nil {
		sampler = sdktrace.TraceIDRatioBased(float64(*options.SampleRatio))
	} else {
		sampler = sdktrace.AlwaysSample()
	}

	if options.Timeout != nil {
		timeout = options.Timeout.AsDuration()
	}

	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(options.HttpEndpoint),
		otlptracehttp.WithTimeout(timeout),
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		log.Fatalf("creating OTLP trace exporter: %v", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
	)

	if options.ServiceName != nil {
		serviceName = *options.ServiceName
	}

	return tp.Tracer(serviceName)
}
