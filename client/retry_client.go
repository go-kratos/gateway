package client

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"strconv"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/endpoint"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/node/direct"
)

type retryClient struct {
	selector selector.Selector

	protocol        config.Protocol
	attempts        uint32
	allowTriedNodes bool
	conditions      [][]uint32
}

func (c *retryClient) Invoke(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	var content []byte
	var selects []string
	// TODO: get fixed bytes from pool if the content-length is specified
	content, err = ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	// copy request to prevent bdoy from being polluted
	req = req.WithContext(ctx)
	req.URL.Scheme = "http"
	req.RequestURI = ""
	req.Body = ioutil.NopCloser(bytes.NewReader(content))

	opts, _ := endpoint.FromContext(ctx)
	filters := opts.Filters
	if !c.allowTriedNodes {
		filter := func(_ context.Context, nodes []selector.Node) []selector.Node {
			if len(selects) == 0 {
				return nodes
			}

			var newNodes []selector.Node
			for _, n := range nodes {
				for _, s := range selects {
					if n.Address() != s {
						newNodes = append(newNodes, n)
					}
				}
			}
			return newNodes
		}
		filters = append(filters, filter)
	}

	for i := 0; i < int(c.attempts); i++ {
		// canceled or deadline exceeded
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}

		selected, done, err := c.selector.Select(ctx, selector.WithFilter(filters...))
		if err != nil {
			break
		}
		wn := selected.(*direct.Node)
		addr := selected.Address()
		selects = append(selects, addr)
		req.URL.Host = selected.Address()
		resp, err = wn.Node.(*node).client.Do(req)
		done(ctx, selector.DoneInfo{Err: err})
		if err != nil {
			continue
		}

		var statusCode uint32
		if c.protocol == config.Protocol_GRPC {
			if resp.StatusCode != 200 {
				continue
			}
			code, _ := strconv.ParseInt(resp.Header.Get("Grpc-Status"), 10, 64)
			statusCode = uint32(code)
		} else {
			statusCode = uint32(resp.StatusCode)
		}
		for _, condition := range c.conditions {
			if len(condition) == 1 {
				if condition[0] == statusCode {
					continue
				} else if statusCode >= condition[0] && statusCode <= condition[1] {
					continue
				}
			}
		}

		// err is nil and no status-conditions is hitted
		break
	}
	return
}
