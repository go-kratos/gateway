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
	// hash得到ID
	id := farm.Fingerprint64([]byte(clientID))
	// 获得rand轮数
	round := int64(id / uint64(count))

	s := rand.NewSource(round)
	ra := rand.New(s)
	//  根据source洗牌
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
	lock     sync.RWMutex
	watcher  map[string]registry.Watcher
	callback map[string]map[string]func([]*registry.ServiceInstance) error
}

func newServiceWatcher() *serviceWatcher {
	return &serviceWatcher{
		watcher:  make(map[string]registry.Watcher),
		callback: make(map[string]map[string]func([]*registry.ServiceInstance) error),
	}
}

func jsonify(in interface{}) string {
	bs, _ := json.Marshal(in)
	return string(bs)
}

func (s *serviceWatcher) Add(ctx context.Context, discovery registry.Discovery, endpoint string, callback func([]*registry.ServiceInstance) error) (watcherExisted bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	LOG.Infof("Add watcher on endpoint: %s", endpoint)
	existed := func() bool {
		ctx, cancel := context.WithTimeout(ctx, time.Second*15)
		defer cancel()
		services, err := discovery.GetService(ctx, endpoint)
		if err != nil {
			LOG.Errorf("Failed to do initial services discovery on endpoint: %s, err: %+v, starting with empty service instance", endpoint, err)
		}
		LOG.Infof("Initialize services discovery on endpoint: %s, services: %s", endpoint, jsonify(services))
		callback(services)

		if _, ok := s.watcher[endpoint]; ok {
			return true
		}
		watcher, err := discovery.Watch(ctx, endpoint)
		if err != nil {
			LOG.Errorf("Failed to initialize watcher on endpoint: %s, err: %+v", endpoint, err)
			return false
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
				if globalSubsetImpl.subsetFn != nil {
					services = globalSubsetImpl.subsetFn(services, globalSubsetImpl.size)
				}
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
	LOG.Infof("Cleanup callback on endpoint: %q with key: %+v", endpoint, cleanup)
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
