package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
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
	parsed struct {
		name   string
		values map[string]struct{}
	}
}

func (c *byHeader) judge(resp *http.Response) bool {
	v := resp.Header.Get(c.ByHeader.Name)
	if v == "" {
		return false
	}
	_, ok := c.parsed.values[v]
	return ok
}

func (c *byHeader) prepare() error {
	c.parsed.name = c.ByHeader.Name
	c.parsed.values = map[string]struct{}{}
	if strings.HasPrefix(c.ByHeader.Value, "[") {
		values, err := parseAsStringList(c.ByHeader.Value)
		if err != nil {
			return err
		}
		for _, v := range values {
			c.parsed.values[v] = struct{}{}
		}
		return nil
	}
	c.parsed.values[c.ByHeader.Value] = struct{}{}
	return nil
}

func parseAsStringList(in string) ([]string, error) {
	out := []string{}
	if err := json.Unmarshal([]byte(in), &out); err != nil {
		return nil, err
	}
	return out, nil
}
