# Gateway
[![Build Status](https://github.com/go-kratos/gateway/workflows/Test/badge.svg?branch=main)](https://github.com/go-kratos/gateway/actions?query=branch%3Amain)
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
