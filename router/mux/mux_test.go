package mux

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestRouter(enabled bool) *muxRouter {
	return NewRouter(
		http.NotFoundHandler(),
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		WithExactFastPath(enabled),
	).(*muxRouter)
}

func TestPathClean(t *testing.T) {
	testCases := []struct {
		Origin   string
		Expected string
	}{
		{
			"/a/b/c",
			"/a/b/c",
		},
		{
			"//a/b",
			"/a/b",
		},
		{
			"/a/b/c/",
			"/a/b/c/",
		},
		{
			"/a//b/c//",
			"/a/b/c/",
		},
	}
	for _, tc := range testCases {
		if cleanPath(tc.Origin) != tc.Expected {
			t.Errorf("cleanPath(%s) %s != %s", tc.Origin, cleanPath(tc.Origin), tc.Expected)
		}
	}
}

func BenchmarkExactPathFastPath(b *testing.B) {
	r := NewRouter(http.NotFoundHandler(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), WithExactFastPath(true))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	if err := r.Handle("/api/v1/room/enter", http.MethodGet, "", handler, noopCloser{}); err != nil {
		b.Fatalf("handle: %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/api/v1/room/enter", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkExactPathMuxFallback(b *testing.B) {
	r := NewRouter(http.NotFoundHandler(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), WithExactFastPath(true))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	if err := r.Handle("/api/v1/room/enter", http.MethodGet, "", handler, noopCloser{}); err != nil {
		b.Fatalf("handle: %v", err)
	}
	// Use a different method to avoid the exact-path fast path.
	req, _ := http.NewRequest(http.MethodPost, "http://example.com/api/v1/room/enter", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkManyRoutesExactPathHit(b *testing.B) {
	r := NewRouter(http.NotFoundHandler(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), WithExactFastPath(true))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	for i := 0; i < 5000; i++ {
		path := "/api/v1/room/" + itoa(i)
		if err := r.Handle(path, http.MethodGet, "", handler, noopCloser{}); err != nil {
			b.Fatalf("handle: %v", err)
		}
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/api/v1/room/1234", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkManyRoutesExactPathMiss(b *testing.B) {
	r := NewRouter(http.NotFoundHandler(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), WithExactFastPath(true))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	for i := 0; i < 5000; i++ {
		path := "/api/v1/room/" + itoa(i)
		if err := r.Handle(path, http.MethodGet, "", handler, noopCloser{}); err != nil {
			b.Fatalf("handle: %v", err)
		}
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/api/v1/room/999999", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func TestExactPathFastPathPrefersEmptyHost(t *testing.T) {
	r := newTestRouter(true)

	hostHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	defaultHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	if err := r.Handle("/api/v1/room/enter", http.MethodGet, "api.example.com", hostHandler, noopCloser{}); err != nil {
		t.Fatalf("handle host route: %v", err)
	}
	if err := r.Handle("/api/v1/room/enter", http.MethodGet, "", defaultHandler, noopCloser{}); err != nil {
		t.Fatalf("handle default route: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/room/enter", nil)
	req.Host = "api.example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("default route status = %d, want %d", w.Code, http.StatusAccepted)
	}

	req = httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/room/enter", nil)
	req.Host = "other.example.com"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("fallback route status = %d, want %d", w.Code, http.StatusAccepted)
	}
}

func TestExactPathFastPathEmptyHostMatchesAnyHost(t *testing.T) {
	r := newTestRouter(true)

	defaultHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	if err := r.Handle("/api/v1/room/enter", http.MethodGet, "", defaultHandler, noopCloser{}); err != nil {
		t.Fatalf("handle default route: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://api.example.com/api/v1/room/enter", nil)
	req.Host = "other.example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("hosted request status = %d, want %d", w.Code, http.StatusAccepted)
	}
}

func TestExactPathFastPathMatchesSpecificHostWithoutDefault(t *testing.T) {
	r := newTestRouter(true)

	hostHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	if err := r.Handle("/api/v1/room/enter", http.MethodGet, "api.example.com", hostHandler, noopCloser{}); err != nil {
		t.Fatalf("handle host route: %v", err)
	}

	h, found := r.exactHandler(http.MethodGet, "/api/v1/room/enter", "api.example.com")
	if !found {
		t.Fatal("exact host route was not found")
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/room/enter", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("host route status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestRouteExactClean(t *testing.T) {
	r := newTestRouter(true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	if err := r.Handle("/api/v1/room/enter", http.MethodGet, "", handler, noopCloser{}); err != nil {
		t.Fatalf("handle route: %v", err)
	}
	if len(r.exact) == 0 {
		t.Fatal("exact fast path should be populated before clean")
	}

	r.RouteExactClean()
	if len(r.exact) != 0 {
		t.Fatalf("exact fast path size = %d, want 0", len(r.exact))
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/room/enter", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("route status after clean = %d, want %d", w.Code, http.StatusAccepted)
	}
}

func TestExactFastPathDisabledByDefault(t *testing.T) {
	r := newTestRouter(false)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	if err := r.Handle("/api/v1/room/enter", http.MethodGet, "", handler, noopCloser{}); err != nil {
		t.Fatalf("handle route: %v", err)
	}
	if len(r.exact) != 0 {
		t.Fatalf("exact fast path size = %d, want 0", len(r.exact))
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/room/enter", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("route status with exact fast path disabled = %d, want %d", w.Code, http.StatusAccepted)
	}
}

func TestExactFastPathEnabledPopulatesExactMap(t *testing.T) {
	r := newTestRouter(true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	if err := r.Handle("/api/v1/room/enter", http.MethodGet, "", handler, noopCloser{}); err != nil {
		t.Fatalf("handle route: %v", err)
	}
	if len(r.exact) == 0 {
		t.Fatal("exact fast path should be populated when enabled")
	}
}

func TestExactFastPathEnabledMatchesCleanPathAndAbsoluteHost(t *testing.T) {
	r := NewRouter(
		http.NotFoundHandler(),
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
		}),
		WithExactFastPath(true),
	)

	hostHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	if err := r.Handle("/api/v1/room/enter", http.MethodGet, "api.example.com", hostHandler, noopCloser{}); err != nil {
		t.Fatalf("handle host route: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://api.example.com//api/v1/room/enter", nil)
	req.Host = "other.example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("route status with cleaned path and absolute host = %d, want %d", w.Code, http.StatusCreated)
	}
}

type noopCloser struct{}

func (noopCloser) Close() error { return nil }

type noopWriter struct{}

func (noopWriter) Header() http.Header         { return http.Header{} }
func (noopWriter) Write(p []byte) (int, error) { return len(p), nil }
func (noopWriter) WriteHeader(statusCode int)  {}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
