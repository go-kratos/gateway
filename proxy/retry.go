package proxy

import (
	"context"
	"net/http"
	"strconv"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/proxy/condition"
)

type retryStrategy struct {
	attempts      int
	timeout       time.Duration
	perTryTimeout time.Duration
	conditions    []condition.Condition
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

func parseRetryConditon(endpoint *config.Endpoint) ([]condition.Condition, error) {
	if endpoint.Retry == nil {
		return []condition.Condition{}, nil
	}
	return condition.ParseConditon(endpoint.Retry.Conditions...)
}

func judgeRetryRequired(conditions []condition.Condition, resp *http.Response) bool {
	return condition.JudgeConditons(conditions, resp, false)
}

var DeriveTimeoutMSHeader = "timeout"

func SetDeriveTimeoutMSHeader(in string) {
	DeriveTimeoutMSHeader = in
}

func setupTimeoutContext(ctx context.Context, req *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	parseMetadataTimeout := func() time.Duration {
		raw := req.Header.Get(DeriveTimeoutMSHeader)
		if raw == "" {
			return 0
		}
		timeoutMs, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return 0
		}
		return time.Millisecond * time.Duration(timeoutMs)
	}
	mdTimeout := parseMetadataTimeout()
	if mdTimeout > 0 {
		if timeout > 0 && mdTimeout < timeout {
			timeout = mdTimeout
		}
	}
	return context.WithTimeout(ctx, timeout)
}
