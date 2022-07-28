package rewrite

import (
	"testing"
)

func BenchmarkStripPrefix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		stripPrefixReqPath(2, "/aa/bb/cc/dd/ee/ff")
	}
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
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := stripPrefixReqPath(tt.stripPrefix, tt.url); got != tt.want {
				t.Errorf("serviceName() = %v, want %v", got, tt.want)
			}
		})
	}
}
