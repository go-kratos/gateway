package client

import (
	"testing"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestCalcAttempts(t *testing.T) {
	testCases := []struct {
		endpoint *config.Endpoint
		attempts int
	}{
		{
			endpoint: &config.Endpoint{
				Retry: &config.Retry{Attempts: 0},
			},
			attempts: 1,
		},
		{
			endpoint: &config.Endpoint{
				Retry: &config.Retry{Attempts: 1},
			},
			attempts: 1,
		},
		{
			endpoint: &config.Endpoint{
				Retry: &config.Retry{Attempts: 2},
			},
			attempts: 2,
		},
		{
			endpoint: &config.Endpoint{},
			attempts: 1,
		},
	}

	for _, testCase := range testCases {
		attempts := calcAttempts(testCase.endpoint)
		if attempts != testCase.attempts {
			t.Errorf("calcAttempts(%v) = %v, want %v", testCase.endpoint, attempts, testCase.attempts)
		}
	}
}

func TestCalcTimeout(t *testing.T) {
	testCase := []struct {
		endpoint *config.Endpoint
		timeout  time.Duration
	}{
		{
			endpoint: &config.Endpoint{
				Timeout: &durationpb.Duration{},
			},
			timeout: time.Second,
		},
		{
			endpoint: &config.Endpoint{
				Timeout: &durationpb.Duration{Seconds: 5},
			},
			timeout: time.Second * 5,
		},
	}

	for _, testCase := range testCase {
		timeout := calcTimeout(testCase.endpoint)
		if timeout != testCase.timeout {
			t.Errorf("calcTimeout(%v) = %v, want %v", testCase.endpoint, timeout, testCase.timeout)
		}
	}
}
