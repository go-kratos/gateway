FROM golang:1.27 AS builder

COPY . /src
WORKDIR /src
RUN mkdir -p bin/
# NOTE: -tags http2legacy is required under the Go 1.27 toolchain because
# golang.org/x/net/http2 (v0.54.0+) wraps the stdlib implementation and no longer
# exports http2.TrailerPrefix, which google.golang.org/grpc still references.
RUN GOPROXY=https://goproxy.cn go build -tags http2legacy -o ./bin/ ./...

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/bin /app

WORKDIR /app

EXPOSE 8080
EXPOSE 7070

VOLUME /data/conf

CMD ["./gateway", "-conf", "/data/conf"]
