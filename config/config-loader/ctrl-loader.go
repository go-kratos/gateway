package ctrlloader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	gorillamux "github.com/gorilla/mux"
	"sigs.k8s.io/yaml"
)

type CtrlConfigLoader struct {
	ctrlService     []string
	ctrlServiceIdx  int
	nextCtrlService bool
	dstPath         string
	cancel          context.CancelFunc

	advertiseName string
	advertiseAddr string
}

type LoadResponse struct {
	Config  string `json:"config"`
	Version string `json:"version"`
}

func prepareCtrlService(in string) []string {
	parts := strings.Split(in, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		u, err := url.Parse(part)
		if err != nil {
			log.Warnf("Failed to parse control service url %s: %s, will skip this one", part, err)
			continue
		}
		out = append(out, u.String())
	}
	if len(out) == 0 {
		log.Warnf("No control service url found, control service will not be available")
	}
	rand.Shuffle(len(out), func(i, j int) {
		out[i], out[j] = out[j], out[i]
	})
	return out
}

func New(name, rawCtrlService, dstPath string) *CtrlConfigLoader {
	cl := &CtrlConfigLoader{
		ctrlService: prepareCtrlService(rawCtrlService),
		dstPath:     dstPath,
	}
	cl.advertiseName = name
	cl.advertiseAddr = cl.getAdvertiseAddr()
	return cl
}

func (c *CtrlConfigLoader) choseCtrlService() string {
	if c.nextCtrlService {
		c.ctrlServiceIdx = (c.ctrlServiceIdx + 1) % len(c.ctrlService)
		c.nextCtrlService = false
		return c.ctrlService[c.ctrlServiceIdx]
	}
	return c.ctrlService[c.ctrlServiceIdx]
}

func (c *CtrlConfigLoader) urlfor(upath string, params url.Values) (string, error) {
	u, err := url.Parse(c.choseCtrlService())
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, upath)
	u.RawQuery = params.Encode()
	return u.String(), nil
}

func (c *CtrlConfigLoader) Load(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			c.nextCtrlService = true
		}
	}()

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
		log.Errorf("%q There was a problem with the IP %+v", c.advertiseName, err)
		return ""
	}
	log.Infof("%s uses IP %s\n", c.advertiseName, advAddr)
	return advAddr
}

func (c *CtrlConfigLoader) load(ctx context.Context) ([]byte, error) {
	params := url.Values{}
	params.Set("gateway", c.advertiseName)
	params.Set("ip_addr", c.advertiseAddr)
	log.Infof("%s is requesting config from %s with params: %+v", c.advertiseName, c.ctrlService, params)
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

type InspectCtrlConfigLoader struct {
	CtrlService     []string `json:"ctrl_service"`
	CtrlServiceIdx  int      `json:"ctrl_service_idx"`
	NextCtrlService bool     `json:"next_ctrl_service"`
	DstPath         string   `json:"dst_path"`
	Hostname        string   `json:"hostname"`
	AdvertiseAddr   string   `json:"advertise_addr"`
}

func (c *CtrlConfigLoader) DebugHandler() http.Handler {
	debugMux := gorillamux.NewRouter()
	debugMux.Methods("GET").Path("/debug/ctrl/inspect").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		out := &InspectCtrlConfigLoader{
			CtrlService:     c.ctrlService,
			CtrlServiceIdx:  c.ctrlServiceIdx,
			NextCtrlService: c.nextCtrlService,
			DstPath:         c.dstPath,
			Hostname:        c.advertiseName,
			AdvertiseAddr:   c.advertiseAddr,
		}
		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(out)
	})
	debugMux.Methods("POST").Path("/debug/ctrl/load").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if err := c.Load(context.Background()); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(struct{}{})
	})
	return debugMux
}
