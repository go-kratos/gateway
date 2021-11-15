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
	parsedCodes []int64
}

func (c *byStatusCode) prepare() error {
	c.parsedCodes = make([]int64, 0, len(c.ByStatusCode))
	parts := strings.Split(c.ByStatusCode, "-")
	if len(parts) == 0 || len(parts) > 2 {
		return fmt.Errorf("invalid condition %s", c.ByStatusCode)
	}
	c.parsedCodes = []int64{}
	for _, p := range parts {
		code, err := strconv.ParseInt(p, 10, 16)
		if err != nil {
			return err
		}
		c.parsedCodes = append(c.parsedCodes, code)
	}
	return nil
}

func (c *byStatusCode) judge(resp *http.Response) bool {
	if len(c.parsedCodes) == 0 {
		return false
	}
	if len(c.parsedCodes) == 1 {
		return int64(resp.StatusCode) == c.parsedCodes[0]
	}
	return (int64(resp.StatusCode) >= c.parsedCodes[0]) &&
		(int64(resp.StatusCode) <= c.parsedCodes[1])
}

type byHeader struct {
	*config.RetryCondition_ByHeader
}

func (c *byHeader) judge(resp *http.Response) bool {
	v := resp.Header.Get(c.ByHeader.Name)
	if v == "" {
		return false
	}
	for _, value := range c.ByHeader.Values {
		if v == value {
			return true
		}
	}
	return false
}

func (c *byHeader) prepare() error {
	name := http.CanonicalHeaderKey(c.ByHeader.Name)
	if name == "Grpc-Status" {
		c.ByHeader.Values = asGrpcNumericCodeValues(c.ByHeader.Values)
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
