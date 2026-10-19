package client

import (
	"context"
	"encoding/json"
	"errors"
	"hash/crc32"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-kratos/gateway/proxy/debug"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/registry"
	"github.com/google/uuid"
)

var ErrCancelWatch = errors.New("cancel watch")
var globalServiceWatcher = newServiceWatcher()
var LOG = log.With("source", "servicewatch")

var _initialResolveTimeout = time.Duration(0)

func init() {
	debug.Register("watcher", globalServiceWatcher)

	func() {
		if timeoutVal := os.Getenv("INITIAL_RESOLVE_TIMEOUT"); timeoutVal != "" {
			initialResolveTimeout, err := time.ParseDuration(timeoutVal)
			if err != nil {
				LOG.Error("Failed to parse INITIAL_RESOLVE_TIMEOUT", "value", timeoutVal, "error", err)
				return
			}
			_initialResolveTimeout = initialResolveTimeout
		}
	}()
}

func uuid4() string {
	return uuid.NewString()
}

func instancesSetHash(instances []*registry.ServiceInstance) string {
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].ID < instances[j].ID
	})
	jsBytes, err := json.Marshal(instances)
	if err != nil {
		return ""
	}
	return strconv.FormatUint(uint64(crc32.ChecksumIEEE(jsBytes)), 10)
}

type watcherStatus struct {
	watcher           registry.Watcher
	initializedChan   chan struct{}
	selectedInstances []*registry.ServiceInstance
}

type serviceWatcher struct {
	lock          sync.RWMutex
	watcherStatus map[string]*watcherStatus
	appliers      map[string]map[string]Applier
}

func newServiceWatcher() *serviceWatcher {
	s := &serviceWatcher{
		watcherStatus: make(map[string]*watcherStatus),
		appliers:      make(map[string]map[string]Applier),
	}
	go s.proccleanup()
	return s
}

func (s *serviceWatcher) setSelectedCache(endpoint string, instances []*registry.ServiceInstance) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.watcherStatus[endpoint].selectedInstances = instances
}

func (s *serviceWatcher) getSelectedCache(endpoint string) ([]*registry.ServiceInstance, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ws, ok := s.watcherStatus[endpoint]
	if ok {
		return ws.selectedInstances, true
	}
	return nil, false
}

func (s *serviceWatcher) getAppliers(endpoint string) (map[string]Applier, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	appliers, ok := s.appliers[endpoint]
	if ok {
		return appliers, true
	}
	return nil, false
}

type Applier interface {
	Callback([]*registry.ServiceInstance) error
	Canceled() bool
}

func (s *serviceWatcher) Add(ctx context.Context, discovery registry.Discovery, endpoint string, applier Applier) (watcherExisted bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	existed := func() bool {
		ws, ok := s.watcherStatus[endpoint]
		if ok {
			// this channel is used to notify the caller that the service watcher is initialized and ready to use
			<-ws.initializedChan

			if len(ws.selectedInstances) > 0 {
				LOG.Info("Using cached selected instances", "count", len(ws.selectedInstances), "endpoint", endpoint, "hash", instancesSetHash(ws.selectedInstances))
				applier.Callback(ws.selectedInstances)
				return true
			}

			return true
		}

		ws = &watcherStatus{
			initializedChan: make(chan struct{}),
		}
		watcher, err := discovery.Watch(ctx, endpoint)
		if err != nil {
			LOG.Error("Failed to initialize watcher", "endpoint", endpoint, "error", err)
			return false
		}
		LOG.Info("Succeeded to initialize watcher", "endpoint", endpoint)
		ws.watcher = watcher
		s.watcherStatus[endpoint] = ws

		func() {
			defer close(ws.initializedChan)
			LOG.Info("Starting to do initialize services discovery", "endpoint", endpoint)

			initialServicesChan := make(chan []*registry.ServiceInstance, 1)
			go func() {
				defer close(initialServicesChan)
				services, err := watcher.Next()
				if err != nil {
					LOG.Error("Failed to do initialize services discovery, the watch process will attempt asynchronously", "endpoint", endpoint, "error", err)
					return
				}
				LOG.Info("Succeeded to do initialize services discovery", "endpoint", endpoint, "services", len(services), "hash", instancesSetHash(ws.selectedInstances))
				initialServicesChan <- services
			}()

			var initialResolveCtx context.Context
			var initialResolveCancel context.CancelFunc
			if _initialResolveTimeout > 0 {
				initialResolveCtx, initialResolveCancel = context.WithTimeout(ctx, _initialResolveTimeout)
			} else {
				initialResolveCtx, initialResolveCancel = context.WithCancel(ctx)
			}
			defer initialResolveCancel()

			select {
			case services := <-initialServicesChan:
				ws.selectedInstances = services
				applier.Callback(services)
			case <-initialResolveCtx.Done():
				emptyServices := []*registry.ServiceInstance{}
				ws.selectedInstances = emptyServices
				applier.Callback(emptyServices)
				LOG.Warn("Initial resolve timeout, will attempt asynchronously", "endpoint", endpoint)
			}
		}()

		go func() {
			for {
				services, err := watcher.Next()
				if err != nil {
					if errors.Is(err, context.Canceled) {
						LOG.Warn("The watch process has been canceled", "endpoint", endpoint)
						return
					}
					LOG.Error("Failed to watch, the watch process will attempt again after 1 second", "endpoint", endpoint, "error", err)
					time.Sleep(time.Second)
					continue
				}
				if len(services) == 0 {
					LOG.Warn("Empty services, this most likely no available instance in discovery", "endpoint", endpoint)
					continue
				}
				LOG.Info("Received services", "count", len(services), "endpoint", endpoint, "hash", instancesSetHash(services))
				s.setSelectedCache(endpoint, services)
				s.doCallback(endpoint, services)
			}
		}()

		return false
	}()

	LOG.Info("Add appliers", "endpoint", endpoint)
	if applier != nil {
		if _, ok := s.appliers[endpoint]; !ok {
			s.appliers[endpoint] = make(map[string]Applier)
		}
		s.appliers[endpoint][uuid4()] = applier
	}

	return existed
}

func (s *serviceWatcher) doCallback(endpoint string, services []*registry.ServiceInstance) {
	canceled := 0
	func() {
		s.lock.RLock()
		defer s.lock.RUnlock()
		for id, applier := range s.appliers[endpoint] {
			if err := applier.Callback(services); err != nil {
				if errors.Is(err, ErrCancelWatch) {
					canceled += 1
					LOG.Warn("applier is canceled, will delete later", "endpoint", endpoint, "id", id)
					continue
				}
				LOG.Error("Failed to call appliers", "endpoint", endpoint, "error", err)
			}
		}
	}()
	if canceled <= 0 {
		return
	}
	LOG.Warn("There are canceled appliers, will be deleted later in cleanup proc", "count", canceled, "endpoint", endpoint)
}

func (s *serviceWatcher) proccleanup() {
	doCleanup := func() {
		for endpoint, appliers := range s.appliers {
			var cleanup []string
			func() {
				s.lock.RLock()
				defer s.lock.RUnlock()
				for id, applier := range appliers {
					if applier.Canceled() {
						cleanup = append(cleanup, id)
						LOG.Warn("applier is canceled, will be deleted later", "endpoint", endpoint, "id", id)
						continue
					}
				}
			}()
			if len(cleanup) <= 0 {
				return
			}
			LOG.Info("Cleanup appliers", "endpoint", endpoint, "keys", cleanup)
			func() {
				s.lock.Lock()
				defer s.lock.Unlock()
				for _, id := range cleanup {
					delete(appliers, id)
				}
				LOG.Info("Succeeded to clean appliers", "cleaned", len(cleanup), "endpoint", endpoint, "available", len(appliers))
			}()
		}
	}

	const interval = time.Second * 30
	for {
		LOG.Info("Start to cleanup appliers on all endpoints", "interval", interval.String())
		time.Sleep(interval)
		doCleanup()
	}
}

func (s *serviceWatcher) DebugHandler() http.Handler {
	debugMux := http.NewServeMux()
	debugMux.HandleFunc("/debug/watcher/nodes", func(w http.ResponseWriter, r *http.Request) {
		service := r.URL.Query().Get("service")
		nodes, _ := s.getSelectedCache(service)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodes)
	})
	debugMux.HandleFunc("/debug/watcher/appliers", func(w http.ResponseWriter, r *http.Request) {
		service := r.URL.Query().Get("service")
		appliers, _ := s.getAppliers(service)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(appliers)
	})
	return debugMux
}

func AddWatch(ctx context.Context, registry registry.Discovery, endpoint string, applier Applier) bool {
	return globalServiceWatcher.Add(ctx, registry, endpoint, applier)
}
