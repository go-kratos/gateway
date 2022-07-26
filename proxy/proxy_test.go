package proxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
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

func BenchmarkStripPrefix(b *testing.B) {
	e := &config.Endpoint{
		StripPrefix: 1,
	}
	req, _ := http.NewRequest("GET", "/aa/bb/cc/dd/ee/ff", strings.NewReader(""))
	for i := 0; i < b.N; i++ {
		stripPrefix(e, req)
	}
}

func stripReq(e *config.Endpoint, req *http.Request) string {
	stripPrefix(e, req)
	return req.URL.Path
}

func TestStripPrefix(t *testing.T) {
	tests := []struct {
		stripPrefix int64
		url         string
		want        string
	}{
		{
			stripPrefix: 1,
			url:         "/a/b/c/d",
			want:        "/b/c/d",
		},
		{
			stripPrefix: 0,
			url:         "/a/b/c/d",
			want:        "/a/b/c/d",
		},
		{
			stripPrefix: -1,
			url:         "/c/a/b/d",
			want:        "/c/a/b/d",
		},
		{
			stripPrefix: 1,
			url:         "/c/a/b/d",
			want:        "/a/b/d",
		},
		{
			stripPrefix: 2,
			url:         "/c/a/b/d",
			want:        "/b/d",
		},
		{
			stripPrefix: 2,
			url:         "/a/b/c/d",
			want:        "/c/d",
		},
		{
			stripPrefix: 3,
			url:         "/a/b/c/d",
			want:        "/d",
		},
		{
			stripPrefix: 4,
			url:         "/a/b/c/d",
			want:        "",
		},
		{
			stripPrefix: 2,
			url:         "/a//b/c/d",
			want:        "/c/d",
		},
		{
			stripPrefix: 2,
			url:         "/a//b//c/d",
			want:        "/c/d",
		},
		{
			stripPrefix: 2,
			url:         "/a//b/c//d",
			want:        "/c/d",
		},
		{
			stripPrefix: 2,
			url:         "/a//b/c//d?a=b&=c",
			want:        "/c/d?a=b&=c",
		},
		{
			stripPrefix: 3,
			url:         "/a//b/c//d?a=b&=c",
			want:        "/d?a=b&=c",
		},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest("get", tt.url, nil)
		t.Run(tt.url, func(t *testing.T) {
			req.URL.Path = tt.url
			if got := stripReq(&config.Endpoint{StripPrefix: tt.stripPrefix}, req); got != tt.want {
				t.Errorf("serviceName() = %v, want %v", got, tt.want)
			}
		})
	}
}
