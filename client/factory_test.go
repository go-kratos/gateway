package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
)

type clientOverrideTransport struct {
	roundTrip func(*http.Request) (*http.Response, error)
	closed    int
}

func (tr *clientOverrideTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return tr.roundTrip(req)
}

func (tr *clientOverrideTransport) CloseIdleConnections() {
	tr.closed++
}

func TestFactoryHTTPClientOverride(t *testing.T) {
	constructors := []struct {
		name string
		new  func(...BuildOption) *BuildContext
	}{
		{"empty", EmptyBuildContext},
		{"configured", func(opts ...BuildOption) *BuildContext {
			return NewBuildContext(&config.Gateway{}, opts...)
		}},
	}
	tests := []struct {
		name     string
		protocol config.Protocol
		tls      bool
		tlsName  string
		host     string
	}{
		{name: "http", protocol: config.Protocol_HTTP},
		{name: "https", protocol: config.Protocol_HTTP, tls: true},
		{name: "https named TLS", protocol: config.Protocol_HTTP, tls: true, tlsName: "unused"},
		{name: "host override", protocol: config.Protocol_HTTP, tls: true, host: "provider.example"},
		{name: "grpc", protocol: config.Protocol_GRPC},
		{name: "grpc TLS", protocol: config.Protocol_GRPC, tls: true},
	}
	for _, constructor := range constructors {
		t.Run(constructor.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					const target = "backend.example:8443"
					endpoint := &config.Endpoint{
						Protocol: tt.protocol,
						Backends: []*config.Backend{{
							Target: target, Tls: tt.tls, TlsConfigName: tt.tlsName,
							Metadata: map[string]string{"host": tt.host},
						}},
					}
					called := false
					tr := &clientOverrideTransport{roundTrip: func(req *http.Request) (*http.Response, error) {
						called = true
						wantScheme, wantHost := "http", "gateway.example"
						if tt.tls {
							wantScheme, wantHost = "https", target
						}
						if tt.host != "" {
							wantHost = tt.host
						}
						if req.URL.Scheme != wantScheme || req.URL.Host != target || req.Host != wantHost {
							t.Errorf("upstream URL = %s, Host = %q; want scheme %q, target %q, Host %q", req.URL, req.Host, wantScheme, target, wantHost)
						}
						if req.RequestURI != "" || req.URL.RequestURI() != "/chat?stream=true" {
							t.Errorf("unexpected request URI: %q, URL: %s", req.RequestURI, req.URL)
						}
						return &http.Response{
							StatusCode: http.StatusCreated,
							Header:     make(http.Header),
							Body:       io.NopCloser(strings.NewReader("response")),
							Request:    req,
						}, nil
					}}
					httpClient := &http.Client{Transport: tr}
					upstream, err := NewFactory(nil)(constructor.new(WithHTTPClient(httpClient)), endpoint)
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { upstream.Close() })
					reqOpts := middleware.NewRequestOptions(endpoint)
					ctx := middleware.NewRequestContext(context.Background(), reqOpts)
					req := httptest.NewRequest(http.MethodGet, "http://gateway.example/chat?stream=true", nil).WithContext(ctx)
					resp, err := upstream.RoundTrip(req)
					if err != nil {
						t.Fatal(err)
					}
					resp.Body.Close()
					reqOpts.DoneFunc(ctx, selector.DoneInfo{})
					if !called || resp.StatusCode != http.StatusCreated {
						t.Fatalf("custom client called = %v, status = %d", called, resp.StatusCode)
					}
					if reqOpts.CurrentNode.Address() != target || len(reqOpts.Backends) != 1 || reqOpts.Backends[0] != target || len(reqOpts.UpstreamResponseTime) != 1 {
						t.Errorf("upstream accounting not preserved: %+v", reqOpts)
					}
					if err := upstream.Close(); err != nil {
						t.Fatal(err)
					}
					if tr.closed != 0 || httpClient.Transport != tr {
						t.Fatal("gateway modified or closed the caller's transport")
					}
					httpClient.CloseIdleConnections()
					if tr.closed != 1 {
						t.Fatal("caller could not close its transport")
					}
				})
			}
		})
	}
}

func TestFactoryHTTPClientOverrideIsolation(t *testing.T) {
	factory := NewFactory(nil)
	endpoint := &config.Endpoint{
		Protocol: config.Protocol_HTTP,
		Backends: []*config.Backend{{Target: "backend.example:443", Tls: true}},
	}
	customClient := &http.Client{}
	for _, tt := range []struct {
		name string
		ctx  *BuildContext
		want *http.Client
	}{
		{"override", EmptyBuildContext(WithHTTPClient(customClient)), customClient},
		{"default", EmptyBuildContext(), _globalHTTPSClient},
		{"nil override", EmptyBuildContext(WithHTTPClient(nil)), _globalHTTPSClient},
	} {
		t.Run(tt.name, func(t *testing.T) {
			upstream, err := factory(tt.ctx, endpoint)
			if err != nil {
				t.Fatal(err)
			}
			defer upstream.Close()
			n, done, err := upstream.(*client).selector.Select(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer done(context.Background(), selector.DoneInfo{})
			if n.(*node).client != tt.want {
				t.Fatal("HTTP client selection leaked between builds")
			}
		})
	}
}

func TestDiscoveryUpdatesKeepHTTPClientOverride(t *testing.T) {
	customClient := &http.Client{}
	endpoint := &config.Endpoint{Protocol: config.Protocol_HTTP}
	upstream, err := NewFactory(nil)(EmptyBuildContext(WithHTTPClient(customClient)), endpoint)
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
