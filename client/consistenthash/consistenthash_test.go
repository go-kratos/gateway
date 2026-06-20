package consistenthash

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/selector"
)

type mockNode struct {
	addr string
}
func (m *mockNode) Scheme() string { return "http" }
func (m *mockNode) Address() string { return m.addr }
func (m *mockNode) ServiceName() string { return "mock" }
func (m *mockNode) InitialWeight() *int64 { return nil }
func (m *mockNode) Version() string { return "v1" }
func (m *mockNode) Metadata() map[string]string { return nil }

func TestConsistentHashSelector(t *testing.T) {
	builder := NewBuilder()
	sel := builder.Build()

	nodes := []selector.Node{
		&mockNode{addr: "node1"},
		&mockNode{addr: "node2"},
		&mockNode{addr: "node3"},
	}

	sel.Apply(nodes)

	ctx1 := WithHashKey(context.Background(), "192.168.1.1")
	node1, _, err := sel.Select(ctx1)
	if err != nil {
		t.Fatalf("expected node, got error: %v", err)
	}

	ctx2 := WithHashKey(context.Background(), "192.168.1.1")
	node2, _, _ := sel.Select(ctx2)
	
	if node1.Address() != node2.Address() {
		t.Fatalf("expected consistent routing for same key, got %v and %v", node1.Address(), node2.Address())
	}

	ctx3 := WithHashKey(context.Background(), "10.0.0.5")
	node3, _, _ := sel.Select(ctx3)
	// Theoretically could be the same, but practically high probability it routes differently or at least consistently for ctx3
	
	ctx4 := WithHashKey(context.Background(), "10.0.0.5")
	node4, _, _ := sel.Select(ctx4)
	if node3.Address() != node4.Address() {
		t.Fatalf("expected consistent routing for same key, got %v and %v", node3.Address(), node4.Address())
	}
}
