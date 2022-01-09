package configloader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"sync"
	"time"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"
)

type OnChange func()

type ConfigLoader interface {
	Pull(context.Context) (*configv1.Gateway, error)
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

func (f *fileLoader) Pull(_ context.Context) (*configv1.Gateway, error) {
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
	f.lock.Lock()
	defer f.lock.Unlock()
	f.onChangeHandlers = append(f.onChangeHandlers, fn)
}

func (f *fileLoader) executeLoader() {
	f.lock.RLock()
	defer f.lock.RUnlock()
	for _, fn := range f.onChangeHandlers {
		fn()
	}
}

func (f *fileLoader) watchproc(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 5):
		}
		sha256hex, err := f.configSHA256()
		if err != nil {
			return
		}
		if sha256hex != f.confSHA256 {
			f.confSHA256 = sha256hex
			f.executeLoader()
		}
	}
}

func (f *fileLoader) Close() {
	f.watchCancel()
}
