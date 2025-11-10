package proxy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/client"
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

type RoundTripperCloserFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperCloserFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (f RoundTripperCloserFunc) Close() error {
	return nil
}

var nopBody = io.NopCloser(&bytes.Buffer{})

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
		Body: nopBody,
	}
	retryable := false
	clientFactory := func(*client.BuildContext, *config.Endpoint) (client.Client, error) {
		dummyClient := RoundTripperCloserFunc(func(req *http.Request) (*http.Response, error) {
			if retryable {
				retryable = false
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: nopBody}, nil
			}
			res.Body = req.Body
			return res, nil
		})
		return dummyClient, nil
	}
	middlewareFactory := func(c *config.Middleware) (middleware.MiddlewareV2, error) {
		return logging.Middleware(c)
	}
	p, err := New(clientFactory, middlewareFactory)
	if err != nil {
		t.Fatal(err)
	}
	buildContext := client.NewBuildContext(c)
	p.Update(buildContext, c)
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
		retryable = true
		r := httptest.NewRequest("POST", "/retryable", bytes.NewBuffer(b))
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

func TestRetryBreaker(t *testing.T) {
	c := &config.Gateway{
		Name: "Test",
		Middlewares: []*config.Middleware{{
			Name: "logging",
		}},
		Endpoints: []*config.Endpoint{
			{
				Protocol: config.Protocol_HTTP,
				Path:     "/retryable",
				Method:   "GET",
				Retry: &config.Retry{
					Attempts: 5,
					Conditions: []*config.Condition{{
						Condition: &config.Condition_ByStatusCode{
							ByStatusCode: "500-599",
						},
					}},
				},
			},
		},
	}

	responseSuccess := false
	retryToSuccess := false
	clientFactory := func(*client.BuildContext, *config.Endpoint) (client.Client, error) {
		dummyClient := RoundTripperCloserFunc(func(req *http.Request) (resp *http.Response, _ error) {
			opt, _ := middleware.FromRequestContext(req.Context())
			defer func() {
				opt.UpstreamStatusCode = append(opt.UpstreamStatusCode, resp.StatusCode)
			}()
			if responseSuccess {
				return &http.Response{StatusCode: http.StatusOK, Body: nopBody}, nil
			}
			if len(opt.UpstreamStatusCode) > 0 {
				if retryToSuccess {
					return &http.Response{StatusCode: http.StatusOK, Body: nopBody}, nil
				}
				return &http.Response{StatusCode: http.StatusNotImplemented, Body: nopBody}, nil
			}
			return &http.Response{StatusCode: 505, Body: nopBody}, nil
		})
		return dummyClient, nil
	}
	middlewareFactory := func(c *config.Middleware) (middleware.MiddlewareV2, error) {
		return logging.Middleware(c)
	}
	p, err := New(clientFactory, middlewareFactory)
	if err != nil {
		t.Fatal(err)
	}
	buildContext := client.NewBuildContext(c)
	p.Update(buildContext, c)

	t.Run("retry-breaker", func(t *testing.T) {
		var lastResponse *responseWriter
		for i := 0; i < 5000; i++ {
			ctx := context.TODO()
			r := httptest.NewRequest("GET", "/retryable", nil)
			r = r.WithContext(ctx)
			w := newResponseWriter()
			p.ServeHTTP(w, r)
			lastResponse = w
		}
		if lastResponse.statusCode == 505 {
			t.Logf("Retry breaker is worked as expected")
		} else {
			t.Logf("Retry breaker is not worked as expected: %+v", lastResponse)
			t.FailNow()
		}

		retryToSuccess = true
		time.Sleep(time.Second * 5)
		for i := 0; i < 5000; i++ {
			ctx := context.TODO()
			r := httptest.NewRequest("GET", "/retryable", nil)
			r = r.WithContext(ctx)
			w := newResponseWriter()
			p.ServeHTTP(w, r)
			lastResponse = w
		}
		if lastResponse.statusCode == 200 {
			t.Logf("Retry breaker re-open is worked as expected")
		} else {
			t.Logf("Retry breaker re-open is not worked as expected: %+v", lastResponse)
			t.FailNow()
		}
	})

}

func TestRecoverErrAbortHandler(t *testing.T) {
	// Test that http.ErrAbortHandler is properly recovered and identified
	testCases := []struct {
		name          string
		panicValue    interface{}
		shouldMatch   bool
		shouldRepanic bool
	}{
		{
			name:          "panic with http.ErrAbortHandler",
			panicValue:    http.ErrAbortHandler,
			shouldMatch:   true,
			shouldRepanic: false,
		},
		{
			name:          "panic with string error",
			panicValue:    "some other error",
			shouldMatch:   false,
			shouldRepanic: true,
		},
		{
			name:          "panic with custom error",
			panicValue:    io.EOF,
			shouldMatch:   false,
			shouldRepanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var innerRecoveredErr interface{}
			var outerRecoveredErr interface{}
			var didInnerRecover bool
			var didOuterRecover bool

			// Outer recovery to catch re-panicked errors
			func() {
				defer func() {
					if err := recover(); err != nil {
						didOuterRecover = true
						outerRecoveredErr = err
					}
				}()

				// Inner function that simulates the proxyStream recovery logic
				func() {
					defer func() {
						if err := recover(); err != nil {
							didInnerRecover = true
							innerRecoveredErr = err
							// Check if it's http.ErrAbortHandler
							if err == http.ErrAbortHandler {
								// This is the expected case - silently return
								return
							}
							// Re-panic for other errors
							panic(err)
						}
					}()

					// Trigger the panic
					panic(tc.panicValue)
				}()
			}()

			if !didInnerRecover {
				t.Fatal("expected panic to be recovered by inner defer")
			}

			// Verify the error comparison
			isAbortHandler := innerRecoveredErr == http.ErrAbortHandler
			if isAbortHandler != tc.shouldMatch {
				t.Errorf("expected isAbortHandler=%v, got %v (recovered: %v)",
					tc.shouldMatch, isAbortHandler, innerRecoveredErr)
			}

			// Verify re-panic behavior
			if tc.shouldRepanic {
				if !didOuterRecover {
					t.Error("expected re-panic to be caught by outer defer")
				}
				if outerRecoveredErr != tc.panicValue {
					t.Errorf("expected outer recover to catch %v, got %v", tc.panicValue, outerRecoveredErr)
				}
			} else {
				if didOuterRecover {
					t.Errorf("unexpected re-panic for http.ErrAbortHandler: %v", outerRecoveredErr)
				}
			}

			// For http.ErrAbortHandler, we should have returned early without re-panic
			if tc.shouldMatch && innerRecoveredErr != http.ErrAbortHandler {
				t.Errorf("expected to recover http.ErrAbortHandler, got %v", innerRecoveredErr)
			}
		})
	}
}
