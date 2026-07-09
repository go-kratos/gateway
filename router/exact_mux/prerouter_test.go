package exact_mux

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// TestRouteOrder documents gorilla/mux's first-match behavior for exact routes
// that share the same path but have different method or host scopes, and
// verifies exactRouter preserves that behavior. Hostless routes match every
// host, and methodless routes match every method, so registering either one
// before a more specific route shadows that more specific route. Registering
// the more specific route first selects it for matching requests.
func TestRouteOrder(t *testing.T) {
	const (
		host = "api.example.com"
		path = "/api/echo"
	)

	testCases := []struct {
		name   string
		routes []testRoute
		method string
		want   string
	}{
		{
			name: "hostless route before host-specific route",
			routes: []testRoute{
				{name: "hostless", method: http.MethodGet},
				{name: "host-specific", method: http.MethodGet, host: host},
			},
			method: http.MethodGet,
			want:   "hostless",
		},
		{
			name: "host-specific route before hostless route",
			routes: []testRoute{
				{name: "host-specific", method: http.MethodGet, host: host},
				{name: "hostless", method: http.MethodGet},
			},
			method: http.MethodGet,
			want:   "host-specific",
		},
		{
			name: "methodless route before method-specific route",
			routes: []testRoute{
				{name: "methodless", host: host},
				{name: "method-specific", method: http.MethodGet, host: host},
			},
			method: http.MethodGet,
			want:   "methodless",
		},
		{
			name: "method-specific route before methodless route",
			routes: []testRoute{
				{name: "method-specific", method: http.MethodGet, host: host},
				{name: "methodless", host: host},
			},
			method: http.MethodGet,
			want:   "method-specific",
		},
		{
			name: "methodless route still handles methods skipped by method-specific route",
			routes: []testRoute{
				{name: "method-specific", method: http.MethodGet, host: host},
				{name: "methodless", host: host},
			},
			method: http.MethodPost,
			want:   "methodless",
		},
	}

	t.Run("gorilla mux", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				router := NewRouter(
					testHandler("not-found"),
					testHandler("method-not-allowed"),
				).(*muxRouter)
				for _, route := range tc.routes {
					mustHandle(t, router, path, route.method, route.host, testHandler(route.name))
				}

				req := httptest.NewRequest(tc.method, path, nil)
				req.Host = host
				if got := serveLabel(router, req); got != tc.want {
					t.Fatalf("handler = %q, want %q", got, tc.want)
				}
			})
		}
	})

	t.Run("exact router", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				router := NewRouter(
					testHandler("not-found"),
					testHandler("method-not-allowed"),
					WithPreRouter(NewExactRouter()),
				).(*muxRouter)
				for _, route := range tc.routes {
					mustHandle(t, router, path, route.method, route.host, testHandler(route.name))
				}

				req := httptest.NewRequest(tc.method, path, nil)
				req.Host = host
				if got := serveLabel(router, req); got != tc.want {
					t.Fatalf("handler = %q, want %q", got, tc.want)
				}
			})
		}
	})
}

type testRoute struct {
	name   string
	method string
	host   string
}

type noopCloser struct{}

func (noopCloser) Close() error { return nil }

func mustHandle(t *testing.T, router *muxRouter, path, method, host string, handler http.Handler) {
	t.Helper()
	if err := router.Handle(path, method, host, handler, noopCloser{}); err != nil {
		t.Fatalf("handle route: %v", err)
	}
}

func testHandler(label string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(label))
	})
}

func serveLabel(handler http.Handler, req *http.Request) string {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w.Body.String()
}

func BenchmarkRouteMatch(b *testing.B) {
	const routeCount = 3000

	testCases := []struct {
		name        string
		exactRouter bool
	}{
		{name: "gorilla mux"},
		{name: "exact router", exactRouter: true},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			router := newBenchmarkRouter(b, routeCount, tc.exactRouter)
			verifyBenchmarkRoutes(b, router, routeCount)

			req := httptest.NewRequest(benchmarkMethod(routeCount-1), benchmarkPath(routeCount-1), nil)
			w := benchmarkResponseWriter{header: make(http.Header)}

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				router.ServeHTTP(&w, req)
			}
		})
	}
}

func newBenchmarkRouter(b *testing.B, routeCount int, exactRouter bool) *muxRouter {
	b.Helper()
	opts := []Option{}
	if exactRouter {
		opts = append(opts, WithPreRouter(NewExactRouter()))
	}
	router := NewRouter(
		benchmarkHandler("not-found"),
		benchmarkHandler("method-not-allowed"),
		opts...,
	).(*muxRouter)
	for i := range routeCount {
		if err := router.Handle(benchmarkPath(i), benchmarkMethod(i), "", benchmarkHandler(benchmarkFlag(i)), noopCloser{}); err != nil {
			b.Fatalf("handle route: %v", err)
		}
	}
	return router
}

func verifyBenchmarkRoutes(b *testing.B, router http.Handler, routeCount int) {
	b.Helper()
	for i := range routeCount {
		w := benchmarkResponseWriter{header: make(http.Header)}
		req := httptest.NewRequest(benchmarkMethod(i), benchmarkPath(i), nil)
		router.ServeHTTP(&w, req)
		if got, want := w.Header().Get(benchmarkRouteHeader), benchmarkFlag(i); got != want {
			b.Fatalf("route %d header %s = %q, want %q", i, benchmarkRouteHeader, got, want)
		}
	}
}

var benchmarkMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
}

func benchmarkMethod(i int) string {
	return benchmarkMethods[i%len(benchmarkMethods)]
}

func benchmarkPath(i int) string {
	return "/api/route/" + strconv.Itoa(i)
}

func benchmarkFlag(i int) string {
	return "route-" + strconv.Itoa(i)
}

const benchmarkRouteHeader = "X-Benchmark-Route"

func benchmarkHandler(flag string) http.Handler {
	value := []string{flag}
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header()[benchmarkRouteHeader] = value
	})
}

type benchmarkResponseWriter struct{ header http.Header }

func (w *benchmarkResponseWriter) Header() http.Header         { return w.header }
func (w *benchmarkResponseWriter) Write(p []byte) (int, error) { return len(p), nil }
func (w *benchmarkResponseWriter) WriteHeader(int)             {}
