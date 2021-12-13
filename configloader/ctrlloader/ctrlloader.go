package ctrlloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
)

type CtrlConfigLoader struct {
	ctrlService string
	dstPath     string
	cancel      context.CancelFunc
}

type LoadResponse struct {
	Config  string `json:"config"`
	Version int64  `json:"version"`
}

func New(ctrlService, dstPath string) *CtrlConfigLoader {
	return &CtrlConfigLoader{
		ctrlService: ctrlService,
		dstPath:     dstPath,
	}
}

func (c *CtrlConfigLoader) urlfor(upath string) (string, error) {
	u, err := url.Parse(c.ctrlService)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, upath)
	return u.String(), nil
}

func (c *CtrlConfigLoader) Load(ctx context.Context) error {
	cfgBytes, err := c.load(ctx)
	if err != nil {
		return err
	}

	resp := &LoadResponse{}
	if err := json.Unmarshal(cfgBytes, &resp); err != nil {
		return err
	}

	tmpPath := fmt.Sprintf("%s.%s.tmp", c.dstPath, uuid.New().String())
	if err := ioutil.WriteFile(tmpPath, []byte(resp.Config), 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, c.dstPath); err != nil {
		return err
	}
	return nil
}

func (c *CtrlConfigLoader) load(ctx context.Context) ([]byte, error) {
	api, err := c.urlfor("/api/v1/config")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid statue code: %d", resp.StatusCode)
	}
	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *CtrlConfigLoader) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	for {
		if err := c.Load(ctx); err != nil {
			// logging
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 5):
		}
	}
}
