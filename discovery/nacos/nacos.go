package nacos

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/go-kratos/gateway/discovery"
	"github.com/go-kratos/kratos/contrib/registry/nacos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"

	// "github.com/hashicorp/consul/api"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func init() {
	discovery.Register("nacos", New)
}

func New(dsn *url.URL) (registry.Discovery, error) {

	h := strings.Split(dsn.Host, ":")
	addr := h[0]
	port := 8848
	if len(h) > 1 {
		uport, _ := strconv.Atoi(h[1])
		port = uport
	}

	sc := []constant.ServerConfig{
		*constant.NewServerConfig(addr, uint64(port)),
	}

	cc := &constant.ClientConfig{
		NamespaceId:         "public",
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
	}

	cli, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  cc,
			ServerConfigs: sc,
		},
	)
	if err != nil {
		log.Error("nacos注册服务失败")
		return nil, err
	}
	r := nacos.New(cli)
	return r, nil
}
