package httputil

import (
	"fmt"
	"sync"
)

type FixedPool struct {
	pool sync.Pool // memory block pool
	size int       // memory block size
}

func NewFixedPool(size int) *FixedPool {
	p := new(FixedPool)
	p.size = size
	return p
}

// GetBlock gets a byte slice from pool
func (p *FixedPool) GetBlock() []byte {
	if v := p.pool.Get(); v != nil {
		return v.([]byte)
	}
	return make([]byte, p.size)
}

// PutBlock releases a byte slice to pool
func (p *FixedPool) PutBlock(block []byte) {
	if len(block) != p.size {
		panic(fmt.Sprintf("block size %d mismatched %d !", len(block), p.size))
	}
	p.pool.Put(block)
}
