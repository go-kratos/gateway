package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

func calcTimeout(endpoint *config.Endpoint) time.Duration {
	var timeout time.Duration
	if endpoint.Timeout != nil {
		timeout = endpoint.Timeout.AsDuration()
	}
	if timeout <= 0 {
		timeout = time.Second
	}
	return timeout
}

func calcAttempts(endpoint *config.Endpoint) int {
	if endpoint.Retry == nil {
		return 1
	}
	if endpoint.Retry.Attempts == 0 {
		return 1
	}
	return int(endpoint.Retry.Attempts)
}

func calcPerTryTimeout(endpoint *config.Endpoint) time.Duration {
	var perTryTimeout time.Duration
	if endpoint.Retry != nil && endpoint.Retry.PerTryTimeout != nil {
		perTryTimeout = endpoint.Retry.PerTryTimeout.AsDuration()
	} else if endpoint.Timeout != nil {
		perTryTimeout = endpoint.Timeout.AsDuration()
	}
	if perTryTimeout <= 0 {
		perTryTimeout = time.Second
	}
	return perTryTimeout
}

func parseRetryConditon(endpoint *config.Endpoint) ([]retryCondition, error) {
	if endpoint.Retry == nil {
		return []retryCondition{}, nil
	}

	conditions := make([]retryCondition, 0, len(endpoint.Retry.Conditions))
	for _, rawCond := range endpoint.Retry.Conditions {
		switch v := rawCond.Condition.(type) {
		case *config.RetryCondition_ByHeader:
			cond := &byHeader{
				RetryCondition_ByHeader: v,
			}
			if err := cond.prepare(); err != nil {
				return nil, err
			}
			conditions = append(conditions, cond)
		case *config.RetryCondition_ByStatusCode:
			cond := &byStatusCode{
				RetryCondition_ByStatusCode: v,
			}
			if err := cond.prepare(); err != nil {
				return nil, err
			}
			conditions = append(conditions, cond)
		default:
			return nil, fmt.Errorf("unknown condition type: %T", v)
		}
	}
	return conditions, nil
}

type retryStrategy struct {
	attempts      int
	timeout       time.Duration
	perTryTimeout time.Duration
	conditions    []retryCondition
}

func prepareRetryStrategy(e *config.Endpoint) (*retryStrategy, error) {
	strategy := &retryStrategy{
		attempts:      calcAttempts(e),
		timeout:       calcTimeout(e),
		perTryTimeout: calcPerTryTimeout(e),
	}
	conditions, err := parseRetryConditon(e)
	if err != nil {
		return nil, err
	}
	strategy.conditions = conditions
	return strategy, nil
}

func judgeRetryRequired(conditions []retryCondition, resp *http.Response) bool {
	for _, cond := range conditions {
		if cond.judge(resp) {
			return true
		}
	}
	return false
}
