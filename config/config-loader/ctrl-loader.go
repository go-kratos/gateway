package ctrlloader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"sigs.k8s.io/yaml"
)

var (
	LOG = log.NewHelper(log.With(log.GetLogger(), "source", "config-loader"))
)

type CtrlConfigLoader struct {
	ctrlService string
	dstPath     string
	cancel      context.CancelFunc

	hostname      string
	advertiseAddr string
}

type LoadResponse struct {
	Config  string `json:"config"`
	Version string `json:"version"`
}

func New(ctrlService, dstPath string) *CtrlConfigLoader {
	cl := &CtrlConfigLoader{
		ctrlService: ctrlService,
		dstPath:     dstPath,
	}
	cl.hostname = cl.getHostname()
	cl.advertiseAddr = cl.getAdvertiseAddr()
	return cl
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

func (c *CtrlConfigLoader) getHostname() string {
	advName := os.Getenv("ADVERTISE_NAME")
	if advName != "" {
		return advName
	}
	hn, _ := os.Hostname()
	return hn
}

func (c *CtrlConfigLoader) getIPInterface(name string) (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Name != name {
			continue // not the name specified
		}

		if iface.Flags&net.FlagUp == 0 {
			return "", errors.New("interfaces is down")
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			return ip.String(), nil
		}
		return "", errors.New("interfaces does not have a valid IPv4")
	}
	return "", errors.New("interface not found")
}

func (c *CtrlConfigLoader) getAdvertiseAddr() string {
	advAddr := os.Getenv("ADVERTISE_ADDR")
	if advAddr != "" {
		return advAddr
	}
	advDevice := os.Getenv("ADVERTISE_DEVICE")
	if advDevice == "" {
		advDevice = "eth0"
	}
	advAddr, err := c.getIPInterface(advDevice)
	if err != nil {
		LOG.Errorf("%q There was a problem with the IP %+v", c.hostname, err)
		return ""
	}
	LOG.Infof("%s uses IP %s\n", c.hostname, advAddr)
	return advAddr
}

func (c *CtrlConfigLoader) load(ctx context.Context) ([]byte, error) {
	params := url.Values{}
	params.Set("gateway", c.hostname)
	params.Set("ip_addr", c.advertiseAddr)
	LOG.Infof("%s is requesting config from %s with params: %+v", c.hostname, c.ctrlService, params)
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
		return nil, fmt.Errorf("invalid status code: %d", resp.StatusCode)
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
