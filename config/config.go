package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/kratos/v2/log"
	gorillamux "github.com/gorilla/mux"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"
)

var (
	LOG = log.NewHelper(log.With(log.GetLogger(), "source", "config"))
)

type OnChange func() error

type ConfigLoader interface {
	Load(context.Context) (*configv1.Gateway, error)
	Watch(OnChange)
	Close()
}

type fileLoader struct {
	confPath         string
	confModTime      int64
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
	modTime, err := f.stat()
	if err != nil {
		return err
	}
	f.confModTime = modTime

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

func (f *fileLoader) stat() (int64, error) {
	fileInfo, err := os.Stat(f.confPath)
	if err != nil {
		return 0, err
	}
	if fileInfo.IsDir() {
		return 0, errors.New("IT’S NOT A FILE")
	}
	return fileInfo.ModTime().Unix(), nil
}

func (f *fileLoader) Load(_ context.Context) (*configv1.Gateway, error) {
	LOG.Infof("loading config file: %s", f.confPath)

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
	LOG.Info("add config file change event handler")
	f.lock.Lock()
	defer f.lock.Unlock()
	f.onChangeHandlers = append(f.onChangeHandlers, fn)
}

func (f *fileLoader) executeLoader() error {
	LOG.Info("execute config loader")
	f.lock.RLock()
	defer f.lock.RUnlock()

	var chainedError error
	for _, fn := range f.onChangeHandlers {
		if err := fn(); err != nil {
			LOG.Errorf("execute config loader error on handler: %+v: %+v", fn, err)
			chainedError = errors.New(err.Error())
		}
	}
	return chainedError
}

func (f *fileLoader) watchproc(ctx context.Context) {
	LOG.Info("start watch config file")
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 5):
		}
		func() {
			modTime, err := f.stat()
			if err != nil {
				LOG.Errorf("watch config file error: %+v", err)
				return
			}
			if modTime != f.confModTime {
				LOG.Infof("config file changed, reload config, last modify time: %s, new modify time: %s", f.confModTime, modTime)
				if err := f.executeLoader(); err != nil {
					LOG.Errorf("execute config loader error with new modify time: %s: %+v, config digest will not be changed until all loaders are succeeded", modTime, err)
					return
				}
				f.confModTime = modTime
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
	ConfModTime      int64  `json:"confModTime"`
	OnChangeHandlers int64  `json:"onChangeHandlers"`
}

func (f *fileLoader) DebugHandler() http.Handler {
	debugMux := gorillamux.NewRouter()
	debugMux.Methods("GET").Path("/debug/config/inspect").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		out := &InspectFileLoader{
			ConfPath:         f.confPath,
			ConfModTime:      f.confModTime,
			OnChangeHandlers: int64(len(f.onChangeHandlers)),
		}
		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(out)
	})
	debugMux.Methods("GET").Path("/debug/config/load").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		out, err := f.Load(context.Background())
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		b, _ := protojson.Marshal(out)
		rw.Write(b)
	})
	debugMux.Methods("GET").Path("/debug/config/version").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		out, err := f.Load(context.Background())
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(map[string]interface{}{
			"version": out.Version,
		})
	})
	return debugMux
}
