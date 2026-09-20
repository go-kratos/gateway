package client

import (
	"net/http"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
)

func TestDefaultNodeClientSelection(t *testing.T) {
	namedClient := &http.Client{}
	for _, tt := range []struct {
		name     string
		protocol config.Protocol
		opts     []NewNodeOption
		want     *http.Client
		tls      bool
	}{
		{"http", config.Protocol_HTTP, nil, _globalClient, false},
		{"grpc", config.Protocol_GRPC, nil, _globalH2CClient, false},
		{"https", config.Protocol_HTTP, []NewNodeOption{WithTLS(true)}, _globalHTTPSClient, true},
		{"grpc TLS", config.Protocol_GRPC, []NewNodeOption{WithTLS(true)}, _globalHTTPSClient, true},
		{"named TLS", config.Protocol_HTTP, []NewNodeOption{WithTLS(true), WithTLSConfigName("named")}, namedClient, true},
		{"missing TLS", config.Protocol_HTTP, []NewNodeOption{WithTLS(true), WithTLSConfigName("missing")}, _globalHTTPSClient, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, explicitNil := range []bool{false, true} {
				var opts []BuildOption
				if explicitNil {
					opts = append(opts, WithHTTPClient(nil))
				}
				ctx := NewBuildContext(&config.Gateway{}, opts...)
				ctx.TLSClientStore.clients["named"] = namedClient
				n := newNode(ctx, "backend.example", tt.protocol, nil, nil, "", "", tt.opts...)
				if n.client != tt.want || n.tls != tt.tls || n.protocol != tt.protocol {
					t.Fatalf("default client selection changed (explicit nil = %v)", explicitNil)
				}
			}
		})
	}
}
