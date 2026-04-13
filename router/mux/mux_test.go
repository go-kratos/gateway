package mux

import (
	"net/http"
	"testing"
)

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
	r := NewRouter(http.NotFoundHandler(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
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
	r := NewRouter(http.NotFoundHandler(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
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
	r := NewRouter(http.NotFoundHandler(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
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
	r := NewRouter(http.NotFoundHandler(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
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
