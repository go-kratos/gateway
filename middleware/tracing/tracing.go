package tracing

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/tracing/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	defaultTimeout     = time.Duration(10 * time.Second)
	defaultServiceName = "gateway"
	defaultTracerName  = "gateway"
)

var globaltp = &struct {
	provider trace.TracerProvider
	initOnce sync.Once
}{}

func init() {
	middleware.Register("tracing", Middleware)
}

// Middleware is a opentelemetry middleware.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.Tracing{}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}
	if globaltp.provider == nil {
		globaltp.initOnce.Do(func() {
			globaltp.provider = newTracerProvider(context.Background(), options)
			propagator := propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})
			otel.SetTracerProvider(globaltp.provider)
			otel.SetTextMapPropagator(propagator)
		})
	}
	tracer := otel.Tracer(defaultTracerName)
	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (reply *http.Response, err error) {
			ctx, span := tracer.Start(
				req.Context(),
				fmt.Sprintf("%s %s", req.Method, req.URL.Path),
				trace.WithSpanKind(trace.SpanKindClient),
			)

			// attributes for each request
			span.SetAttributes(
				semconv.HTTPMethodKey.String(req.Method),
				semconv.HTTPTargetKey.String(req.URL.Path),
				semconv.NetPeerIPKey.String(req.RemoteAddr),
			)

			defer func() {
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
				} else {
					span.SetStatus(codes.Ok, "OK")
				}
				if reply != nil {
					span.SetAttributes(semconv.HTTPStatusCodeKey.Int(reply.StatusCode))
				}
				span.End()
			}()
			return next.RoundTrip(req.WithContext(ctx))
		})
	}, nil
}

func newTracerProvider(ctx context.Context, options *v1.Tracing) trace.TracerProvider {
	var (
		timeout     = defaultTimeout
		serviceName = defaultServiceName
	)

	if appInfo, ok := kratos.FromContext(ctx); ok {
		serviceName = appInfo.Name()
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

	// attributes for all requests
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
