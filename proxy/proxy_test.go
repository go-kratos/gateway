package proxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/gateway/middleware/logging"
)

type responseWriter struct {
	statusCode int
	header     http.Header
	body       bytes.Buffer
}

func (r *responseWriter) Header() http.Header {
	return r.header
}

func (r *responseWriter) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func newResponseWriter() *responseWriter {
	return &responseWriter{header: make(http.Header)}
}

func TestProxy(t *testing.T) {
	c := &config.Gateway{
		Name: "Test",
		Middlewares: []*config.Middleware{{
			Name: "logging",
		}},
		Endpoints: []*config.Endpoint{{
			Protocol: config.Protocol_HTTP,
			Path:     "/foo/bar",
			Method:   "GET",
		}, {
			Protocol: config.Protocol_HTTP,
			Path:     "/retryable",
			Method:   "POST",
			Retry: &config.Retry{
				Attempts: 3,
				Conditions: []*config.Condition{{
					Condition: &config.Condition_ByStatusCode{
						ByStatusCode: "500-504",
					},
				}},
			},
		}},
	}
	res := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"testKey": []string{"testValue"},
		},
	}
	shouldRetry := "should-retry"
	clientFactory := func(*config.Endpoint) (http.RoundTripper, error) {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get(shouldRetry) == "true" {
				req.Header.Del(shouldRetry)
				return &http.Response{StatusCode: http.StatusInternalServerError}, nil
			}
			res.Body = req.Body
			return res, nil
		}), nil
	}
	middlewareFactory := func(c *config.Middleware) (middleware.Middleware, error) {
		return logging.Middleware(c)
	}
	p, err := New(clientFactory, middlewareFactory)
	if err != nil {
		t.Fatal(err)
	}
	p.Update(c)
	{
		b := []byte("notfound")
		r := httptest.NewRequest("GET", "/notfound", bytes.NewBuffer(b))
		w := newResponseWriter()
		p.ServeHTTP(w, r)
		if w.statusCode != http.StatusNotFound {
			t.Fatalf("want ok but got: %+v", w)
		}
	}
	{
		b := []byte("ok")
		r := httptest.NewRequest("GET", "/foo/bar", bytes.NewBuffer(b))
		w := newResponseWriter()
		p.ServeHTTP(w, r)
		if w.statusCode != res.StatusCode {
			t.Fatalf("want ok but got: %+v", w)
		}
		if !reflect.DeepEqual(w.header, res.Header) {
			t.Fatalf("want %+v but got %+v", res.Header, w.header)
		}
		if !bytes.Equal(b, w.body.Bytes()) {
			t.Fatalf("want %+v but got %+v", b, w.body.Bytes())
		}
	}
	{
		b := []byte("retryable")
		r := httptest.NewRequest("POST", "/retryable", bytes.NewBuffer(b))
		r.Header.Set(shouldRetry, "true")
		w := newResponseWriter()
		p.ServeHTTP(w, r)
		if w.statusCode != res.StatusCode {
			t.Fatalf("want ok but got: %+v", w)
		}
		if !reflect.DeepEqual(w.header, res.Header) {
			t.Fatalf("want %+v but got %+v", res.Header, w.header)
		}
		if !bytes.Equal(b, w.body.Bytes()) {
			t.Fatalf("want %+v but got %+v", b, w.body.Bytes())
		}
	}
}
