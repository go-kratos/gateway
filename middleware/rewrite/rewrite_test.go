package rewrite

import "testing"

func TestStripPrefix(t *testing.T) {
	p1 := "/dddd/"
	p1Cases := map[string]string{
		"/dddd/api":      "/api",
		"/dddd/dddd/api": "/dddd/api",
	}

	p2 := "/dddd"
	p2Cases := map[string]string{
		"/dddd/api":      "/api",
		"/dddd/dddd/api": "/dddd/api",
	}

	for k, v := range p1Cases {
		t.Logf("stripPrefix(%s, %s) = %s", k, p1, v)
		if stripPrefix(k, p1) != v {
			t.Errorf("stripPrefix(%s, %s) != %s", k, p1, v)
		}
	}

	for k, v := range p2Cases {
		t.Logf("stripPrefix(%s, %s) = %s", k, p1, v)
		if stripPrefix(k, p2) != v {
			t.Errorf("stripPrefix(%s, %s) != %s", k, p2, v)
		}
	}
}
