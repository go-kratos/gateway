package condition

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
)

type Condition interface {
	Prepare() error
	Judge(*http.Response) bool
}

type byStatusCode struct {
	*config.Condition_ByStatusCode
	parsedCodes []int64
}

func (c *byStatusCode) Prepare() error {
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

func (c *byStatusCode) Judge(resp *http.Response) bool {
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
	*config.Condition_ByHeader
	parsed struct {
		name   string
		values map[string]struct{}
	}
}

func (c *byHeader) Judge(resp *http.Response) bool {
	v := resp.Header.Get(c.ByHeader.Name)
	if v == "" {
		return false
	}
	_, ok := c.parsed.values[v]
	return ok
}

func (c *byHeader) Prepare() error {
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

func ParseConditon(in ...*config.Condition) ([]Condition, error) {
	conditions := make([]Condition, 0, len(in))
	for _, rawCond := range in {
		switch v := rawCond.Condition.(type) {
		case *config.Condition_ByHeader:
			cond := &byHeader{
				Condition_ByHeader: v,
			}
			if err := cond.Prepare(); err != nil {
				return nil, err
			}
			conditions = append(conditions, cond)
		case *config.Condition_ByStatusCode:
			cond := &byStatusCode{
				Condition_ByStatusCode: v,
			}
			if err := cond.Prepare(); err != nil {
				return nil, err
			}
			conditions = append(conditions, cond)
		default:
			return nil, fmt.Errorf("unknown condition type: %T", v)
		}
	}
	return conditions, nil
}

func JudgeConditons(conditions []Condition, resp *http.Response, onEmpty bool) bool {
	if len(conditions) <= 0 {
		return onEmpty
	}
	for _, cond := range conditions {
		if cond.Judge(resp) {
			return true
		}
	}
	return false
}
