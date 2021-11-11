package client

import (
	"testing"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestParseRetryConditon(t *testing.T) {
	testCases := []struct {
		endpoint   *config.Endpoint
		conditions [][]uint32
	}{
		{
			endpoint: &config.Endpoint{
				Protocol: config.Protocol_HTTP,
				Retry: &config.Retry{
					Conditions: []string{"500"},
				},
			},
			conditions: [][]uint32{{500}},
		},
		{
			endpoint: &config.Endpoint{
				Protocol: config.Protocol_HTTP,
				Retry: &config.Retry{
					Conditions: []string{"501", "502"},
				},
			},
			conditions: [][]uint32{{501}, {502}},
		},
		{
			endpoint: &config.Endpoint{
				Protocol: config.Protocol_HTTP,
				Retry: &config.Retry{
					Conditions: []string{"400-500", "501"},
				},
			},
			conditions: [][]uint32{{400, 500}, {501}},
		},
		{
			endpoint: &config.Endpoint{
				Protocol: config.Protocol_GRPC,
				Retry: &config.Retry{
					Conditions: []string{`"NOT_FOUND"`, `"CANCELLED"`},
				},
			},
			conditions: [][]uint32{{5}, {1}},
		},
	}

	for _, testCase := range testCases {
		conditions, err := parseRetryConditon(testCase.endpoint)
		if err != nil {
			t.Fatal(err)
		}
		if len(conditions) != len(testCase.conditions) {
			t.Errorf("parseRetryConditon(%v) = %v, want %v", testCase.endpoint, conditions, testCase.conditions)
		}
		for i, condition := range conditions {
			if condition[0] != testCase.conditions[i][0] {
				t.Errorf("parseRetryConditon(%v) = %v, want %v", testCase.endpoint, conditions, testCase.conditions)
			}
		}
	}
}

func TestCalcAttempts(t *testing.T) {
	testCases := []struct {
		endpoint *config.Endpoint
		attempts uint32
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
				Timeout: &durationpb.Duration{Seconds: 1},
			},
			timeout: time.Second,
		},
		{
			endpoint: &config.Endpoint{
				Timeout: &durationpb.Duration{Seconds: 5},
				Retry:   &config.Retry{PerTryTimeout: &durationpb.Duration{Seconds: 2}},
			},
			timeout: time.Second * 2,
		},
	}

	for _, testCase := range testCase {
		timeout := calcTimeout(testCase.endpoint)
		if timeout != testCase.timeout {
			t.Errorf("calcTimeout(%v) = %v, want %v", testCase.endpoint, timeout, testCase.timeout)
		}
	}
}
