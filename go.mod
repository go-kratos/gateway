module github.com/go-kratos/gateway

go 1.15

require (
	github.com/go-kratos/kratos/contrib/registry/consul/v2 v2.0.0-20211104134829-037296cdbf54
	github.com/go-kratos/kratos/v2 v2.1.2
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/consul/api v1.11.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v1.0.1
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.0.1
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.0.1
	go.opentelemetry.io/otel/sdk v1.0.1
	go.opentelemetry.io/otel/trace v1.0.1
	go.uber.org/automaxprocs v1.4.0
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	google.golang.org/grpc v1.42.0
	google.golang.org/grpc/examples v0.0.0-20211119181224-d542bfcee46d
	google.golang.org/protobuf v1.27.1
)

replace github.com/go-kratos/kratos/v2 v2.1.2 => github.com/go-kratos/kratos/v2 v2.0.0-20211204183355-63a7ffae0487
