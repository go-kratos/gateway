package mux

import (
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
