package client

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"google.golang.org/grpc/codes"
)

type retryCondition interface {
	prepare() error
	judge(*http.Response) bool
}

type byStatusCode struct {
	*config.RetryCondition_ByStatusCode
	parsedCodes [][]int64
}

func (c *byStatusCode) prepare() error {
	c.parsedCodes = make([][]int64, 0, len(c.ByStatusCode.StatusCodes))
	for _, raw := range c.ByStatusCode.StatusCodes {
		cs := strings.Split(raw, "-")
		if len(cs) == 0 || len(cs) > 2 {
			return fmt.Errorf("invalid condition %s", raw)
		}
		condCodes := []int64{}
		for _, c := range cs {
			code, err := strconv.ParseInt(c, 10, 16)
			if err != nil {
				return err
			}
			condCodes = append(condCodes, code)
		}
		c.parsedCodes = append(c.parsedCodes, condCodes)
	}
	return nil
}

func (c *byStatusCode) judge(resp *http.Response) bool {
	if len(c.parsedCodes) == 0 {
		return false
	}
	for _, condCodes := range c.parsedCodes {
		if len(condCodes) == 1 {
			if int64(resp.StatusCode) == condCodes[0] {
				return true
			}
			continue
		}
		if (int64(resp.StatusCode) >= condCodes[0]) &&
			(int64(resp.StatusCode) <= condCodes[1]) {
			return true
		}
	}
	return false
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

func (c *byHeader) prepare() error {
	for _, header := range c.RetryCondition_ByHeader.ByHeader.Headers {
		name := http.CanonicalHeaderKey(header.Name)
		if name == "Grpc-Status" {
			header.Values = asGrpcNumericCodeValues(header.Values)
		}
	}
	return nil
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
