package otel

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/otel/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestTracer(t *testing.T) {
	cfg, err := anypb.New(&v1.Otel{
		HttpEndpoint: "127.0.0.1:4318",
	})
	assert.NoError(t, err, "new any pb error")

	next := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		return &http.Response{
			Body: ioutil.NopCloser(bytes.NewBufferString("Hello Kratos")),
		}, nil
	}
	ctx := context.Background()

	m, err := Middleware(context.Background(), &config.Middleware{
		Options: cfg,
	})
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/v1/hello", bytes.NewBufferString("test"))
	_, err = m(next)(ctx, req)
	assert.NoError(t, err)
}
