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
	"sigs.k8s.io/yaml"
)

type CtrlConfigLoader struct {
	ctrlService string
	dstPath     string
	cancel      context.CancelFunc
}

type LoadResponse struct {
	Config  string `json:"config"`
	Version string `json:"version"`
}

func New(ctrlService, dstPath string) *CtrlConfigLoader {
	return &CtrlConfigLoader{
		ctrlService: ctrlService,
		dstPath:     dstPath,
	}
}

func (c *CtrlConfigLoader) urlfor(upath string, params url.Values) (string, error) {
	u, err := url.Parse(c.ctrlService)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, upath)
	u.RawQuery = params.Encode()
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

	yamlBytes, err := yaml.JSONToYAML([]byte(resp.Config))
	if err != nil {
		return err
	}

	tmpPath := fmt.Sprintf("%s.%s.tmp", c.dstPath, uuid.New().String())
	if err := ioutil.WriteFile(tmpPath, yamlBytes, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, c.dstPath); err != nil {
		return err
	}
	return nil
}

func hostname() string {
	hn, _ := os.Hostname()
	hn = "test-gw"
	return hn
}

func advertiseAddr() string {
	return "192.168.1.10"
}

func (c *CtrlConfigLoader) load(ctx context.Context) ([]byte, error) {
	params := url.Values{}
	params.Set("gateway", hostname())
	params.Set("ip_addr", advertiseAddr())
	api, err := c.urlfor("/v1/control/gateway/release", params)
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
