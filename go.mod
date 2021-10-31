module github.com/go-kratos/gateway

go 1.15

require (
	github.com/go-kratos/kratos/contrib/registry/consul/v2 v2.0.0-20211010065212-69fc5cca876c
	github.com/go-kratos/kratos/v2 v2.0.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/consul/api v1.11.0
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	google.golang.org/grpc v1.39.1
	google.golang.org/protobuf v1.27.1
)

replace github.com/go-kratos/kratos/v2 v2.0.0 => github.com/go-kratos/kratos/v2 v2.0.0-20211010065212-69fc5cca876c
