package client

import (
	"context"
	"net"
	"testing"

	"github.com/go-kratos/kratos/v2/selector"
)

type mockPicker struct {
	appliedNodes []selector.Node
}

func (m *mockPicker) Apply(nodes []selector.Node) {
	m.appliedNodes = nodes
}
func (m *mockPicker) Select(ctx context.Context, opts ...selector.SelectOption) (selector.Node, selector.DoneFunc, error) {
	return nil, nil, nil
}

type mockNode struct {
	addr string
}
func (m *mockNode) Scheme() string { return "tcp" }
func (m *mockNode) Address() string { return m.addr }
func (m *mockNode) ServiceName() string { return "mock" }
func (m *mockNode) InitialWeight() *int64 { return nil }
func (m *mockNode) Version() string { return "v1" }
func (m *mockNode) Metadata() map[string]string { return nil }

func TestHealthChecker(t *testing.T) {
	// Start a dummy TCP server to represent a healthy node
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock tcp server: %v", err)
	}
	defer listener.Close()

	healthyAddr := listener.Addr().String()
	unhealthyAddr := "127.0.0.1:9999" // Assuming nothing listens here

	picker := &mockPicker{}
	hc := NewHealthChecker(picker)
	defer hc.Stop()

	nodes := []selector.Node{
		&mockNode{addr: healthyAddr},
		&mockNode{addr: unhealthyAddr},
	}

	hc.UpdateNodes(nodes)

	if len(picker.appliedNodes) != 2 {
		t.Fatalf("expected 2 nodes applied initially, got %d", len(picker.appliedNodes))
	}

	// Force checks
	hc.checkAll() // failure count = 1 for unhealthy
	hc.checkAll() // failure count = 2 for unhealthy, should be removed

	if len(picker.appliedNodes) != 1 {
		t.Fatalf("expected 1 node applied after failures, got %d", len(picker.appliedNodes))
	}
	if picker.appliedNodes[0].Address() != healthyAddr {
		t.Fatalf("expected healthy node %s, got %s", healthyAddr, picker.appliedNodes[0].Address())
	}
}
