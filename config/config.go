package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/kratos/v2/log"
	gorillamux "github.com/gorilla/mux"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"
)

type OnChange func() error

type ConfigLoader interface {
	Load(context.Context) (*configv1.Gateway, error)
	Watch(OnChange)
	Close()
}

type fileLoader struct {
	confPath         string
	confSHA256       string
	watchCancel      context.CancelFunc
	lock             sync.RWMutex
	onChangeHandlers []OnChange
}

var _jsonOptions = &protojson.UnmarshalOptions{DiscardUnknown: true}

func NewFileLoader(confPath string) (ConfigLoader, error) {
	fl := &fileLoader{
		confPath: confPath,
	}
	if err := fl.initialize(); err != nil {
		return nil, err
	}
	return fl, nil
}

func (f *fileLoader) initialize() error {
	sha256hex, err := f.configSHA256()
	if err != nil {
		return err
	}
	f.confSHA256 = sha256hex
	log.Infof("the initial config file sha256: %s", sha256hex)

	watchCtx, cancel := context.WithCancel(context.Background())
	f.watchCancel = cancel
	go f.watchproc(watchCtx)
	return nil
}

func sha256sum(in []byte) string {
	sum := sha256.Sum256(in)
	return hex.EncodeToString(sum[:])
}

func (f *fileLoader) configSHA256() (string, error) {
	configData, err := ioutil.ReadFile(f.confPath)
	if err != nil {
		return "", err
	}
	return sha256sum(configData), nil
}

func (f *fileLoader) Load(_ context.Context) (*configv1.Gateway, error) {
	log.Infof("loading config file: %s", f.confPath)

	configData, err := ioutil.ReadFile(f.confPath)
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
	return out, nil
}

func (f *fileLoader) Watch(fn OnChange) {
	log.Info("add config file change event handler")
	f.lock.Lock()
	defer f.lock.Unlock()
	f.onChangeHandlers = append(f.onChangeHandlers, fn)
}

func (f *fileLoader) executeLoader() error {
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

func (f *fileLoader) watchproc(ctx context.Context) {
	log.Info("start watch config file")
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 5):
		}
		func() {
			sha256hex, err := f.configSHA256()
			if err != nil {
				log.Errorf("watch config file error: %+v", err)
				return
			}
			if sha256hex != f.confSHA256 {
				log.Infof("config file changed, reload config, last sha256: %s, new sha256: %s", f.confSHA256, sha256hex)
				if err := f.executeLoader(); err != nil {
					log.Errorf("execute config loader error with new sha256: %s: %+v, config digest will not be changed until all loaders are succeeded", sha256hex, err)
					return
				}
				f.confSHA256 = sha256hex
				return
			}
		}()
	}
}

func (f *fileLoader) Close() {
	f.watchCancel()
}

type InspectFileLoader struct {
	ConfPath         string `json:"confPath"`
	ConfSHA256       string `json:"confSha256"`
	OnChangeHandlers int64  `json:"onChangeHandlers"`
}

func (f *fileLoader) DebugHandler() http.Handler {
	debugMux := gorillamux.NewRouter()
	debugMux.Methods("GET").Path("/debug/config/inspect").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		out := &InspectFileLoader{
			ConfPath:         f.confPath,
			ConfSHA256:       f.confSHA256,
			OnChangeHandlers: int64(len(f.onChangeHandlers)),
		}
		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(out)
	})
	debugMux.Methods("GET").Path("/debug/config/load").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
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
	debugMux.Methods("GET").Path("/debug/config/version").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
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
