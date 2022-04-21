module github.com/go-kratos/gateway

go 1.15

require (
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13
	github.com/fatih/color v1.10.0 // indirect
	github.com/go-kratos/aegis v0.1.1
	github.com/go-kratos/kratos/contrib/registry/consul/v2 v2.0.0-20220318065833-e66a2905ab70
	github.com/go-kratos/kratos/v2 v2.2.1
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/consul/api v1.12.0
	github.com/kr/text v0.2.0 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/prometheus/client_golang v1.12.1
	go.opentelemetry.io/otel v1.4.1
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.4.1
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.4.1
	go.opentelemetry.io/otel/sdk v1.4.1
	go.opentelemetry.io/otel/trace v1.4.1
	go.uber.org/automaxprocs v1.4.0
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd
	google.golang.org/grpc v1.44.0
	google.golang.org/grpc/examples v0.0.0-20220223153006-a73725f42db9
	google.golang.org/protobuf v1.27.1
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	sigs.k8s.io/yaml v1.3.0
)
