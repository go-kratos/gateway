module github.com/go-kratos/gateway

go 1.15

require (
	github.com/go-kratos/aegis v0.1.1 // indirect
	github.com/go-kratos/kratos/contrib/registry/consul/v2 v2.0.0-20211010065212-69fc5cca876c
	github.com/go-kratos/kratos/v2 v2.0.0
	github.com/go-playground/form/v4 v4.2.0 // indirect
	github.com/golang-jwt/jwt/v4 v4.0.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/consul/api v1.11.0
	go.opentelemetry.io/otel/sdk v1.0.0 // indirect
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	google.golang.org/genproto v0.0.0-20210805201207-89edb61ffb67 // indirect
	google.golang.org/grpc v1.39.1
	google.golang.org/protobuf v1.27.1
)

replace (
	github.com/go-kratos/kratos/v2 v2.0.0 =>	github.com/go-kratos/kratos/v2 v2.0.0-20211010065212-69fc5cca876c
)