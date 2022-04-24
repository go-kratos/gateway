package client

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/dgryski/go-farm"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/google/uuid"
)

var ErrCancelWatch = errors.New("cancel watch")
var globalServiceWatcher = newServiceWatcher()

const defaultSubsetSize = 20

type subsetFn func(instances []*registry.ServiceInstance, size int) []*registry.ServiceInstance

var globalSubsetImpl = &struct {
	subsetFn subsetFn
	size     int
}{
	subsetFn: defaultSubset,
	size:     defaultSubsetSize,
}

func genClientID() string {
	hostname := os.Getenv("HOSTNAME")
	if hostname != "" {
		return hostname
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = strconv.Itoa(int(time.Now().UnixNano()))
	}
	return hostname
}

func defaultSubset(instances []*registry.ServiceInstance, size int) []*registry.ServiceInstance {
	backends := instances
	if size <= 0 {
		return backends
	}
	if len(backends) <= int(size) {
		return backends
	}
	clientID := genClientID()
	sort.Slice(backends, func(i, j int) bool {
		return backends[i].ID < backends[j].ID
	})
	count := len(backends) / size
	id := farm.Fingerprint64([]byte(clientID))
	round := int64(id / uint64(count))

	s := rand.NewSource(round)
	ra := rand.New(s)
	ra.Shuffle(len(backends), func(i, j int) {
		backends[i], backends[j] = backends[j], backends[i]
	})
	start := (id % uint64(count)) * uint64(size)
	return backends[int(start) : int(start)+int(size)]
}

func uuid4() string {
	return uuid.NewString()
}

type serviceWatcher struct {
	lock              sync.RWMutex
	watcher           map[string]registry.Watcher
	selectedInstances map[string][]*registry.ServiceInstance
	callback          map[string]map[string]func([]*registry.ServiceInstance) error
}

func newServiceWatcher() *serviceWatcher {
	return &serviceWatcher{
		watcher:           make(map[string]registry.Watcher),
		callback:          make(map[string]map[string]func([]*registry.ServiceInstance) error),
		selectedInstances: make(map[string][]*registry.ServiceInstance),
	}
}

func jsonify(in interface{}) string {
	bs, _ := json.Marshal(in)
	return string(bs)
}

func (s *serviceWatcher) pickAndCacheSelected(endpoint string, instances []*registry.ServiceInstance) []*registry.ServiceInstance {
	if globalSubsetImpl.subsetFn != nil {
		instances = globalSubsetImpl.subsetFn(instances, globalSubsetImpl.size)
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.selectedInstances[endpoint] = instances
	return instances
}

func (s *serviceWatcher) getSelectedCache(endpoint string) ([]*registry.ServiceInstance, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	si, ok := s.selectedInstances[endpoint]
	if ok {
		return si, true
	}
	return nil, false
}

func (s *serviceWatcher) Add(ctx context.Context, discovery registry.Discovery, endpoint string, callback func([]*registry.ServiceInstance) error) (watcherExisted bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	existed := func() bool {
		if si, ok := s.selectedInstances[endpoint]; ok {
			LOG.Infof("Using cached %d selected instances on endpoint: %s ", len(si), endpoint)
			callback(si)
			return true
		}

		watcher, err := discovery.Watch(ctx, endpoint)
		if err != nil {
			LOG.Errorf("Failed to initialize watcher on endpoint: %s, err: %+v", endpoint, err)
			return false
		}
		LOG.Infof("Succeeded to initialize watcher on endpoint: %s", endpoint)
		s.watcher[endpoint] = watcher

		go func() {
			for {
				services, err := watcher.Next()
				if err != nil {
					if errors.Is(err, context.Canceled) {
						LOG.Warnf("The watch process on: %s has been canceled", endpoint, err)
						return
					}
					LOG.Errorf("Failed to watch on endpoint: %s, err: %+v, the watch process will attempt again after 1 second", endpoint, err)
					time.Sleep(time.Second)
					continue
				}
				if len(services) == 0 {
					LOG.Warnf("Empty services on endpoint: %s, this most likely no available instance in discovery", endpoint)
					continue
				}
				services = s.pickAndCacheSelected(endpoint, services)
				s.doCallback(endpoint, services)
			}
		}()

		return false
	}()

	LOG.Infof("Add callback on endpoint: %s", endpoint)
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
					LOG.Warnf("callback on endpoint: %s, id: %s is canceled, will delete later", endpoint, id)
					continue
				}
				LOG.Errorf("Failed to call callback on endpoint: %q: %+v", endpoint, err)
			}
		}
	}()

	if len(cleanup) <= 0 {
		return
	}
	LOG.Infof("Cleanup callback on endpoint: %q with keys: %+v", endpoint, cleanup)
	func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		for _, id := range cleanup {
			delete(s.callback[endpoint], id)
		}
	}()
}

func AddWatch(ctx context.Context, registry registry.Discovery, endpoint string, callback func([]*registry.ServiceInstance) error) bool {
	return globalServiceWatcher.Add(ctx, registry, endpoint, callback)
}

func setGlobalSubsetImpl(fn subsetFn, size int) {
	globalSubsetImpl.subsetFn = fn
	globalSubsetImpl.size = size
}
