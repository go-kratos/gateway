module github.com/go-kratos/gateway

go 1.15

require (
	github.com/go-kratos/aegis v0.1.2
	github.com/go-kratos/kratos/contrib/registry/consul/v2 v2.0.0-20220318065833-e66a2905ab70
	github.com/go-kratos/kratos/v2 v2.2.2
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/consul/api v1.12.0
	github.com/prometheus/client_golang v1.12.1
	go.opentelemetry.io/otel v1.4.1
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.4.1
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.4.1
	go.opentelemetry.io/otel/sdk v1.4.1
	go.opentelemetry.io/otel/trace v1.4.1
	go.uber.org/automaxprocs v1.4.0
	golang.org/x/net v0.0.0-20220513224357-95641704303c
	google.golang.org/genproto v0.0.0-20220401170504-314d38edb7de
	google.golang.org/protobuf v1.28.0
	sigs.k8s.io/yaml v1.3.0
)
