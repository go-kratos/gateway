package client

import (
	"context"
	"errors"
	"sync"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/google/uuid"
)

var ErrCancelWatch = errors.New("cancel watch")

var globalServiceWatcher = &serviceWatcher{
	watcher:  make(map[string]registry.Watcher),
	callback: make(map[string]map[string]func([]*registry.ServiceInstance) error),
}

func uuid4() string {
	return uuid.NewString()
}

type serviceWatcher struct {
	lock     sync.RWMutex
	watcher  map[string]registry.Watcher
	callback map[string]map[string]func([]*registry.ServiceInstance) error
}

func (s *serviceWatcher) Add(endpoint string, watcher registry.Watcher, callback func([]*registry.ServiceInstance) error) (watcherExisted bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	existed := func() bool {
		if _, ok := s.watcher[endpoint]; ok {
			return true
		}
		s.watcher[endpoint] = watcher

		go func() {
			for {
				services, err := watcher.Next()
				if err != nil && errors.Is(err, context.Canceled) {
					return
				}
				if len(services) == 0 {
					continue
				}
				s.doCallback(endpoint, services)
			}
		}()

		return false
	}()

	if callback != nil {
		if _, ok := s.callback[endpoint]; !ok {
			s.callback[endpoint] = make(map[string]func([]*registry.ServiceInstance) error)
		}
		s.callback[endpoint][uuid4()] = callback
	}

	return existed
}

func (s *serviceWatcher) doCallback(endpoint string, services []*registry.ServiceInstance) {
	cleanup := []string{}

	func() {
		s.lock.RLock()
		defer s.lock.RUnlock()
		for id, callback := range s.callback[endpoint] {
			if err := callback(services); err != nil {
				if errors.Is(err, ErrCancelWatch) {
					cleanup = append(cleanup, id)
				}
			}
		}
	}()

	if len(cleanup) <= 0 {
		return
	}
	func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		for _, id := range cleanup {
			delete(s.callback[endpoint], id)
		}
	}()
}

func AddWatch(endpoint string, watcher registry.Watcher, callback func([]*registry.ServiceInstance) error) bool {
	return globalServiceWatcher.Add(endpoint, watcher, callback)
}
