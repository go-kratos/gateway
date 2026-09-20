# Gateway
[![Build Status](https://github.com/go-kratos/gateway/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/go-kratos/gateway/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/go-kratos/gateway/branch/main/graph/badge.svg)](https://codecov.io/gh/go-kratos/gateway)

HTTP -> Proxy -> Router -> Middleware -> Client -> Selector -> Node

## Protocol
* HTTP -> HTTP  
* HTTP -> gRPC  
* gRPC -> gRPC  

## Encoding
* Protobuf Schemas

## Endpoint
* prefix: /api/echo/*
* path: /api/echo/hello
* regex: /api/echo/[a-z]+
* restful: /api/echo/{name}

## Middleware
* cors
* auth
* color
* logging
* tracing
* metrics
* ratelimit
* datacenter

## Custom upstream HTTP clients

Supply a custom `*http.Client` for a single client build through a build option:

```go
buildCtx := client.EmptyBuildContext(client.WithHTTPClient(httpClient))
upstream, err := clientFactory(buildCtx, endpoint)
```

`client.NewBuildContext(gatewayConfig, client.WithHTTPClient(httpClient))` also
accepts the option. The override applies to all backends in that build, including
nodes added by service discovery. Omitting the option or passing `nil` preserves
the existing HTTP, HTTPS, and gRPC client selection.

The supplied client replaces the complete HTTP client, including its transport,
TLS configuration, redirect policy, and timeout. Backend configuration still
controls the request scheme and host; named TLS configurations are not merged
into the supplied client. Configure the client before use and reuse its transport
where appropriate. The caller owns its lifecycle: closing a gateway client does
not close the supplied client's idle connections.
