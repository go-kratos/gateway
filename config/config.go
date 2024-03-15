package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"
)

type OnChange func() error

type ConfigLoader interface {
	Load(context.Context) (*configv1.Gateway, error)
	Watch(OnChange)
	Close()
}

type FileLoader struct {
	confPath           string
	confSHA256         string
	priorityDirectory  string
	priorityConfigHash map[string]string
	watchCancel        context.CancelFunc
	lock               sync.RWMutex
	onChangeHandlers   []OnChange
}

var _jsonOptions = &protojson.UnmarshalOptions{DiscardUnknown: true}

func NewFileLoader(confPath string, priorityDirectory string) (*FileLoader, error) {
	fl := &FileLoader{
		confPath:          confPath,
		priorityDirectory: priorityDirectory,
	}
	if err := fl.initialize(); err != nil {
		return nil, err
	}
	return fl, nil
}

func (f *FileLoader) initialize() error {
	if f.priorityDirectory != "" {
		if err := os.MkdirAll(f.priorityDirectory, 0755); err != nil {
			return err
		}
	}
	sha256hex, pfHash, err := f.configSHA256()
	if err != nil {
		return err
	}
	f.confSHA256 = sha256hex
	log.Infof("the initial config file sha256: %s", sha256hex)
	f.priorityConfigHash = pfHash
	log.Infof("the initial priority config file sha256 map: %+v", f.priorityConfigHash)

	watchCtx, cancel := context.WithCancel(context.Background())
	f.watchCancel = cancel
	go f.watchproc(watchCtx)
	return nil
}

func sha256sum(in []byte) string {
	sum := sha256.Sum256(in)
	return hex.EncodeToString(sum[:])
}

func (f *FileLoader) configSHA256() (string, map[string]string, error) {
	configData, err := os.ReadFile(f.confPath)
	if err != nil {
		return "", nil, err
	}
	hash := sha256sum(configData)
	phHash, err := f.priorityConfigSHA256()
	if err != nil {
		log.Warnf("failed to get priority config sha256: %+v", err)
	}
	return hash, phHash, nil
}

func (f *FileLoader) priorityConfigSHA256() (map[string]string, error) {
	if f.priorityDirectory == "" {
		return map[string]string{}, nil
	}
	entrys, err := os.ReadDir(f.priorityDirectory)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, e := range entrys {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		configData, err := os.ReadFile(filepath.Join(f.priorityDirectory, e.Name()))
		if err != nil {
			return nil, err
		}
		out[e.Name()] = sha256sum(configData)
	}
	return out, nil
}

func (f *FileLoader) Load(_ context.Context) (*configv1.Gateway, error) {
	log.Infof("loading config file: %s", f.confPath)

	configData, err := os.ReadFile(f.confPath)
	if err != nil {
		return nil, err
	}

	jsonData, err := yaml.YAMLToJSON(configData)
	if err != nil {
		return nil, err
	}
	out := &configv1.Gateway{}
	if err := _jsonOptions.Unmarshal(jsonData, out); err != nil {
		return nil, err
	}
	if err := f.mergePriorityConfig(out); err != nil {
		log.Warnf("failed to merge priority config: %+v", err)
	}
	return out, nil
}

func (f *FileLoader) mergePriorityConfig(dst *configv1.Gateway) error {
	if f.priorityDirectory == "" {
		return nil
	}
	entrys, err := os.ReadDir(f.priorityDirectory)
	if err != nil {
		return err
	}
	replaceOrAppendEndpoint := MakeReplaceOrAppendEndpointFn(dst.Endpoints)
	for _, e := range entrys {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		cfgPath := filepath.Join(f.priorityDirectory, e.Name())
		pCfg, err := f.parsePriorityConfig(cfgPath)
		if err != nil {
			log.Warnf("failed to parse priority config: %s: %+v, skip merge this file", cfgPath, err)
			continue
		}
		for _, e := range pCfg.Endpoints {
			dst.Endpoints = replaceOrAppendEndpoint(dst.Endpoints, e)
		}
		log.Infof("succeeded to merge priority config: %s, %d endpoints effected", cfgPath, len(pCfg.Endpoints))
	}
	return nil
}

func (f *FileLoader) parsePriorityConfig(cfgPath string) (*configv1.PriorityConfig, error) {
	configData, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	jsonData, err := yaml.YAMLToJSON(configData)
	if err != nil {
		return nil, err
	}
	out := &configv1.PriorityConfig{}
	if err := _jsonOptions.Unmarshal(jsonData, out); err != nil {
		return nil, err
	}
	return out, nil
}

func MakeReplaceOrAppendEndpointFn(origin []*configv1.Endpoint) func([]*configv1.Endpoint, *configv1.Endpoint) []*configv1.Endpoint {
	keyFn := func(e *configv1.Endpoint) string {
		return fmt.Sprintf("%s-%s", e.Method, e.Path)
	}
	index := map[string]int{}
	for i, e := range origin {
		index[keyFn(e)] = i
	}
	return func(dst []*configv1.Endpoint, item *configv1.Endpoint) []*configv1.Endpoint {
		idx, ok := index[keyFn(item)]
		if !ok {
			return append(dst, item)
		}
		dst[idx] = item
		return dst
	}
}

func (f *FileLoader) Watch(fn OnChange) {
	log.Info("add config file change event handler")
	f.lock.Lock()
	defer f.lock.Unlock()
	f.onChangeHandlers = append(f.onChangeHandlers, fn)
}

func (f *FileLoader) executeLoader() error {
	log.Info("execute config loader")
	f.lock.RLock()
	defer f.lock.RUnlock()

	var chainedError error
	for _, fn := range f.onChangeHandlers {
		if err := fn(); err != nil {
			log.Errorf("execute config loader error on handler: %+v: %+v", fn, err)
			chainedError = errors.New(err.Error())
		}
	}
	return chainedError
}

func (f *FileLoader) watchproc(ctx context.Context) {
	log.Info("start watch config file")
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 5):
		}
		func() {
			sha256hex, pfHash, err := f.configSHA256()
			if err != nil {
				log.Errorf("watch config file error: %+v", err)
				return
			}
			if sha256hex != f.confSHA256 || !reflect.DeepEqual(pfHash, f.priorityConfigHash) {
				log.Infof("config file changed, reload config, last sha256: %s, new sha256: %s, last pfHash: %+v, new pfHash: %+v", f.confSHA256, sha256hex, f.priorityConfigHash, pfHash)
				if err := f.executeLoader(); err != nil {
					log.Errorf("execute config loader error with new sha256: %s: %+v, config digest will not be changed until all loaders are succeeded", sha256hex, err)
					return
				}
				f.confSHA256 = sha256hex
				f.priorityConfigHash = pfHash
				return
			}
		}()
	}
}

func (f *FileLoader) Close() {
	f.watchCancel()
}

type InspectFileLoader struct {
	ConfPath           string            `json:"confPath"`
	ConfSHA256         string            `json:"confSha256"`
	PriorityConfigHash map[string]string `json:"priorityConfigHash"`
	OnChangeHandlers   int64             `json:"onChangeHandlers"`
}

func (f *FileLoader) DebugHandler() http.Handler {
	debugMux := http.NewServeMux()
	debugMux.HandleFunc("/debug/config/inspect", func(rw http.ResponseWriter, r *http.Request) {
		out := &InspectFileLoader{
			ConfPath:           f.confPath,
			ConfSHA256:         f.confSHA256,
			PriorityConfigHash: f.priorityConfigHash,
			OnChangeHandlers:   int64(len(f.onChangeHandlers)),
		}
		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(out)
	})
	debugMux.HandleFunc("/debug/config/load", func(rw http.ResponseWriter, r *http.Request) {
		out, err := f.Load(context.Background())
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(err.Error()))
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		b, _ := protojson.Marshal(out)
		_, _ = rw.Write(b)
	})
	debugMux.HandleFunc("/debug/config/version", func(rw http.ResponseWriter, r *http.Request) {
		out, err := f.Load(context.Background())
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(err.Error()))
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(map[string]interface{}{
			"version": out.Version,
		})
	})
	return debugMux
}
