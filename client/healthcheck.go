package client

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/selector"
)

// HealthChecker actively monitors the health of backend nodes.
type HealthChecker struct {
	picker      selector.Selector
	nodes       []selector.Node
	healthy     map[string]bool
	failures    map[string]int
	mu          sync.RWMutex
	cancel      context.CancelFunc
}

// NewHealthChecker creates a new HealthChecker that updates the given selector.
func NewHealthChecker(picker selector.Selector) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	hc := &HealthChecker{
		picker:   picker,
		healthy:  make(map[string]bool),
		failures: make(map[string]int),
		cancel:   cancel,
	}
	go hc.loop(ctx)
	return hc
}

// UpdateNodes updates the list of nodes to monitor and applies healthy ones to the picker.
func (hc *HealthChecker) UpdateNodes(nodes []selector.Node) {
	hc.mu.Lock()
	hc.nodes = nodes
	// Initialize new nodes as healthy by default to allow traffic immediately.
	for _, n := range nodes {
		if _, exists := hc.healthy[n.Address()]; !exists {
			hc.healthy[n.Address()] = true
			hc.failures[n.Address()] = 0
		}
	}
	hc.mu.Unlock()
	hc.applyHealthy()
}

// Stop stops the background health check loop.
func (hc *HealthChecker) Stop() {
	hc.cancel()
}

func (hc *HealthChecker) loop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.checkAll()
		}
	}
}

func (hc *HealthChecker) checkAll() {
	hc.mu.RLock()
	nodes := hc.nodes
	hc.mu.RUnlock()

	changed := false

	for _, n := range nodes {
		addr := n.Address()
		// Simple TCP ping
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		
		hc.mu.Lock()
		isCurrentlyHealthy := hc.healthy[addr]
		
		if err != nil {
			hc.failures[addr]++
			if hc.failures[addr] >= 2 && isCurrentlyHealthy {
				log.Warnf("healthcheck failed for %s, removing from active pool", addr)
				hc.healthy[addr] = false
				changed = true
			}
		} else {
			conn.Close()
			hc.failures[addr] = 0
			if !isCurrentlyHealthy {
				log.Infof("healthcheck recovered for %s, adding back to active pool", addr)
				hc.healthy[addr] = true
				changed = true
			}
		}
		hc.mu.Unlock()
	}

	if changed {
		hc.applyHealthy()
	}
}

func (hc *HealthChecker) applyHealthy() {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var active []selector.Node
	for _, n := range hc.nodes {
		if hc.healthy[n.Address()] {
			active = append(active, n)
		}
	}
	hc.picker.Apply(active)
}
