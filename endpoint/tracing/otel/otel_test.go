package otel

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/tracing/v1"
	"github.com/go-kratos/gateway/endpoint"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
)

type mockRequest struct {
}

func (r *mockRequest) Host() string {
	return ""
}

func (r *mockRequest) Path() string {
	return ""
}

func (r *mockRequest) Method() string {
	return ""
}

func (r *mockRequest) Query() url.Values {
	return nil
}

func (r *mockRequest) Header() http.Header {
	return nil
}

func (r *mockRequest) Body() io.ReadCloser {
	return nil
}

type mockResponse struct {
}

func TestTracer(t *testing.T) {
	serviceName := "kratos gateway"
	cfg, err := anypb.New(&v1.Tracing{
		HttpEndpoint: "127.0.0.1:1234",
		ServiceName:  &serviceName,
	})
	assert.NoError(t, err, "new any pb error")

	next := func(ctx context.Context, req endpoint.Request) (endpoint.Response, error) {
		return nil, nil
	}
	ctx := context.Background()

	m, err := Middleware(&config.Middleware{
		Options: cfg,
	})
	assert.NoError(t, err)

	_, err = m(next)(ctx, nil)
	assert.NoError(t, err)
}
