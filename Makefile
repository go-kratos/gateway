API_PROTO_FILES=$(shell find api -name *.proto)

# http2legacy is required under the Go 1.27 toolchain: golang.org/x/net/http2
# (v0.54.0+) wraps the stdlib implementation and drops http2.TrailerPrefix,
# which google.golang.org/grpc still references.
GO_BUILD_TAGS?=http2legacy

.PHONY: api build test vet
# generate api proto
api:
	protoc --proto_path=./api \
 	       --go_out=paths=source_relative:./api \
	       $(API_PROTO_FILES)

build:
	go build -tags $(GO_BUILD_TAGS) -o ./bin/ ./...

test:
	go test -tags $(GO_BUILD_TAGS) ./...

vet:
	go vet -tags $(GO_BUILD_TAGS) ./...
