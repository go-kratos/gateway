package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
)

func TestFactoryHTTPClientOverride(t *testing.T) {
	endpoint := &config.Endpoint{
		Protocol: config.Protocol_HTTP,
		Backends: []*config.Backend{{
			Target: "backend.example:8443", Tls: true, TlsConfigName: "unused",
		}},
	}
	customClient := &http.Client{Transport: middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Scheme != "https" {
			t.Errorf("scheme = %q, want https", req.URL.Scheme)
		}
		return &http.Response{StatusCode: http.StatusCreated, Body: http.NoBody}, nil
	})}
	upstream, err := NewFactory(nil)(EmptyBuildContext(WithHTTPClient(customClient)), endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer upstream.Close()

	reqOpts := middleware.NewRequestOptions(endpoint)
	ctx := middleware.NewRequestContext(context.Background(), reqOpts)
	req := httptest.NewRequest(http.MethodGet, "http://gateway.example/", nil).WithContext(ctx)
	resp, err := upstream.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	reqOpts.DoneFunc(ctx, selector.DoneInfo{})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
}

func TestDiscoveryUpdatesKeepHTTPClientOverride(t *testing.T) {
	customClient := &http.Client{}
	endpoint := &config.Endpoint{Protocol: config.Protocol_HTTP}
	upstream, err := NewFactory(nil)(NewBuildContext(&config.Gateway{}, WithHTTPClient(customClient)), endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer upstream.Close()
	c := upstream.(*client)
	for _, address := range []string{"first.example:8080", "second.example:8080"} {
		if err := c.applier.Callback([]*registry.ServiceInstance{{
			Name: "provider", Endpoints: []string{"http://" + address},
		}}); err != nil {
			t.Fatal(err)
		}
		n, done, err := c.selector.Select(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		done(context.Background(), selector.DoneInfo{})
		if n.Address() != address || n.(*node).client != customClient {
			t.Fatalf("discovery update did not retain override for %s", address)
		}
	}
}
