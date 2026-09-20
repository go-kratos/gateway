package client

import (
	"net/http"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
)

func TestNodeHTTPClientSelection(t *testing.T) {
	namedClient := &http.Client{}
	customClient := &http.Client{}
	for _, tt := range []struct {
		name     string
		protocol config.Protocol
		tls      bool
		tlsName  string
		override *http.Client
		want     *http.Client
	}{
		{"http", config.Protocol_HTTP, false, "", nil, _globalClient},
		{"grpc", config.Protocol_GRPC, false, "", nil, _globalH2CClient},
		{"https", config.Protocol_HTTP, true, "", nil, _globalHTTPSClient},
		{"grpc TLS", config.Protocol_GRPC, true, "", nil, _globalHTTPSClient},
		{"named TLS", config.Protocol_HTTP, true, "named", nil, namedClient},
		{"missing TLS", config.Protocol_HTTP, true, "missing", nil, _globalHTTPSClient},
		{"http override", config.Protocol_HTTP, false, "", customClient, customClient},
		{"grpc override", config.Protocol_GRPC, false, "", customClient, customClient},
		{"TLS override", config.Protocol_HTTP, true, "named", customClient, customClient},
		{"grpc TLS override", config.Protocol_GRPC, true, "", customClient, customClient},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewBuildContext(&config.Gateway{}, WithHTTPClient(tt.override))
			ctx.TLSClientStore.clients["named"] = namedClient
			n := newNode(ctx, "backend.example", tt.protocol, nil, nil, "", "", WithTLS(tt.tls), WithTLSConfigName(tt.tlsName))
			if n.client != tt.want {
				t.Errorf("client = %p, want %p", n.client, tt.want)
			}
			if n.tls != tt.tls {
				t.Errorf("TLS = %v, want %v", n.tls, tt.tls)
			}
		})
	}
}
