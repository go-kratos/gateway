module github.com/go-kratos/gateway

go 1.15

require (
	github.com/go-kratos/kratos/contrib/registry/consul/v2 v2.0.0-20220124072645-9ea78f302d5a
	github.com/go-kratos/kratos/v2 v2.1.5
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/consul/api v1.12.0
	github.com/prometheus/client_golang v1.12.0
	go.opentelemetry.io/otel v1.3.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.3.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.3.0
	go.opentelemetry.io/otel/sdk v1.3.0
	go.opentelemetry.io/otel/trace v1.3.0
	go.uber.org/automaxprocs v1.4.0
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd
	google.golang.org/grpc v1.44.0
	google.golang.org/grpc/examples v0.0.0-20220125225548-e27717498dbc
	google.golang.org/protobuf v1.27.1
	sigs.k8s.io/yaml v1.3.0
)
