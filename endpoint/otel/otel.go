package otel

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/otel/v1"
	"github.com/go-kratos/gateway/endpoint"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const Name = "opentelemetery"

var globaltp = &struct {
	provider trace.TracerProvider
	initOnce sync.Once
}{}

func Middleware(cfg *config.Middleware) (endpoint.Middleware, error) {
	options := &v1.Otel{}
	if err := cfg.Options.UnmarshalTo(options); err != nil {
		return nil, errors.WithStack(err)
	}

	if globaltp.provider == nil {
		globaltp.initOnce.Do(func() {
			globaltp.provider = NewTracerProvider(context.Background(), options)
			propagator := propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})
			otel.SetTracerProvider(globaltp.provider)
			otel.SetTextMapPropagator(propagator)
		})
	}

	tracer := otel.Tracer("gateway")

	return func(handler endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req *http.Request) (reply *http.Response, err error) {
			ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", req.Method, req.URL.Path), trace.WithSpanKind(trace.SpanKindClient))
			span.SetAttributes(
				semconv.HTTPMethodKey.String(req.Method),
				semconv.HTTPTargetKey.String(req.URL.Path),
			)
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

func NewTracerProvider(ctx context.Context, options *v1.Otel) trace.TracerProvider {
	var (
		serviceName = "gateway"
		timeout     = time.Duration(10 * time.Second)
	)

	if options.ServiceName != nil {
		serviceName = *options.ServiceName
	}

	if options.Timeout != nil {
		timeout = options.Timeout.AsDuration()
	}

	var sampler sdktrace.Sampler
	if options.SampleRatio == nil {
		sampler = sdktrace.AlwaysSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(float64(*options.SampleRatio))
	}

	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(options.HttpEndpoint),
		otlptracehttp.WithTimeout(timeout),
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		log.Fatalf("creating OTLP trace exporter: %v", err)
	}

	resources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
	)

	return sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resources),
	)
}
