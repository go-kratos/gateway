package otel

import (
	"io"
	"net/http"
	"net/url"
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

// func TestTracer(t *testing.T) {
// 	serviceName := "kratos gateway"
// 	cfg, err := anypb.New(&v1.Tracing{
// 		HttpEndpoint: "127.0.0.1:1234",
// 		ServiceName:  &serviceName,
// 	})
// 	assert.NoError(t, err, "new any pb error")

// 	next := func(ctx context.Context, req *http.Request) (*http.Response, error) {
// 		return nil, nil
// 	}
// 	ctx := context.Background()

// 	m, err := Middleware(&config.Middleware{
// 		Options: cfg,
// 	})
// 	assert.NoError(t, err)

// 	_, err = m(next)(ctx, nil)
// 	assert.NoError(t, err)
// }
