package client

import (
	"net/http"
	"strconv"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"google.golang.org/grpc/codes"
)

type retryCondition interface {
	prepare()
	judge(*http.Response) bool
}

type byStatusCode struct {
	*config.RetryCondition_ByStatusCode
}

func (c *byStatusCode) prepare() {}

func (c *byStatusCode) judge(resp *http.Response) bool {
	if len(c.ByStatusCode.StatusCodes) == 0 {
		return false
	}
	if len(c.ByStatusCode.StatusCodes) == 1 {
		return resp.StatusCode == int(c.ByStatusCode.StatusCodes[0])
	}
	return (resp.StatusCode >= int(c.ByStatusCode.StatusCodes[0])) &&
		(resp.StatusCode <= int(c.ByStatusCode.StatusCodes[1]))
}

type byHeader struct {
	*config.RetryCondition_ByHeader
}

func (c *byHeader) judge(resp *http.Response) bool {
	for _, header := range c.ByHeader.Headers {
		v := resp.Header.Get(header.Name)
		if v == "" {
			continue
		}
		for _, value := range header.Values {
			if v == value {
				return true
			}
		}
	}
	return false
}

func (c *byHeader) prepare() {
	for _, header := range c.RetryCondition_ByHeader.ByHeader.Headers {
		name := http.CanonicalHeaderKey(header.Name)
		if name == "Grpc-Status" {
			header.Values = asGrpcNumericCodeValues(header.Values)
		}
	}
}

func asGrpcNumericCodeValues(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		code, ok := asGrpcCode(v)
		if !ok {
			continue
		}
		out = append(out, strconv.FormatInt(int64(code), 10))
	}
	return out
}

func asGrpcCode(in string) (codes.Code, bool) {
	c := codes.Code(0)
	if err := c.UnmarshalJSON([]byte(in)); err != nil {
		// logging
		return codes.Code(0), false
	}
	return c, true
}
