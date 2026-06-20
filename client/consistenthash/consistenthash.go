package consistenthash

import (
	"context"
	"hash/crc32"
	"sort"
	"strconv"
	"sync"

	"github.com/go-kratos/kratos/v2/selector"
)

const Name = "consistenthashing"

// DefaultReplicas is the default number of virtual nodes per physical node.
const DefaultReplicas = 100

type hashKey struct{}

// WithHashKey injects the routing hash key into the context.
func WithHashKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, hashKey{}, key)
}

// HashKeyFromContext extracts the routing hash key from the context.
func HashKeyFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(hashKey{}).(string); ok {
		return val
	}
	return ""
}

type builder struct {
	replicas int
}

// NewBuilder creates a consistent hashing selector builder.
func NewBuilder() selector.Builder {
	return &builder{replicas: DefaultReplicas}
}

func (b *builder) Build() selector.Selector {
	return &consistentHashSelector{
		replicas: b.replicas,
	}
}

type consistentHashSelector struct {
	replicas int
	mu       sync.RWMutex
	keys     []uint32
	hashMap  map[uint32]selector.Node
	nodes    []selector.Node
}

func (s *consistentHashSelector) Apply(nodes []selector.Node) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nodes = nodes
	s.keys = nil
	s.hashMap = make(map[uint32]selector.Node)

	for _, node := range nodes {
		for i := 0; i < s.replicas; i++ {
			hash := crc32.ChecksumIEEE([]byte(node.Address() + strconv.Itoa(i)))
			s.keys = append(s.keys, hash)
			s.hashMap[hash] = node
		}
	}
	sort.Slice(s.keys, func(i, j int) bool {
		return s.keys[i] < s.keys[j]
	})
}

func (s *consistentHashSelector) Select(ctx context.Context, opts ...selector.SelectOption) (selector.Node, selector.DoneFunc, error) {
	var options selector.SelectOptions
	for _, opt := range opts {
		opt(&options)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.keys) == 0 {
		return nil, nil, selector.ErrNoAvailable
	}

	keyStr := HashKeyFromContext(ctx)

	if keyStr == "" {
		// fallback to first node if no key provided
		return s.nodes[0], func(context.Context, selector.DoneInfo) {}, nil
	}

	hash := crc32.ChecksumIEEE([]byte(keyStr))

	// Binary search for appropriate replica
	idx := sort.Search(len(s.keys), func(i int) bool {
		return s.keys[i] >= hash
	})

	if idx == len(s.keys) {
		idx = 0
	}

	node := s.hashMap[s.keys[idx]]
	
	// Apply filters if any
	for _, filter := range options.NodeFilters {
		filtered := filter(ctx, []selector.Node{node})
		if len(filtered) == 0 {
			// For simplicity, we just return nil if filtered
			return nil, nil, selector.ErrNoAvailable
		}
	}

	return node, func(context.Context, selector.DoneInfo) {}, nil
}
