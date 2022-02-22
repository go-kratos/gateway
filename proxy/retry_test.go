package proxy

import (
	"net/http"
	"testing"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestRetryByStatusCode(t *testing.T) {
	testCases := []struct {
		cond   *byStatusCode
		resp   *http.Response
		result bool
	}{
		{
			cond: &byStatusCode{
				RetryCondition_ByStatusCode: &config.RetryCondition_ByStatusCode{
					ByStatusCode: "501",
				},
			},
			resp:   &http.Response{StatusCode: 501},
			result: true,
		},
		{
			cond: &byStatusCode{
				RetryCondition_ByStatusCode: &config.RetryCondition_ByStatusCode{
					ByStatusCode: "501-509",
				},
			},
			resp:   &http.Response{StatusCode: 500},
			result: false,
		},
		{
			cond: &byStatusCode{
				RetryCondition_ByStatusCode: &config.RetryCondition_ByStatusCode{
					ByStatusCode: "501-509",
				},
			},
			resp:   &http.Response{StatusCode: 502},
			result: true,
		},
	}

	for _, testCase := range testCases {
		if err := testCase.cond.prepare(); err != nil {
			t.Errorf("prepare error: %v", err)
		}
		result := testCase.cond.judge(testCase.resp)
		if result != testCase.result {
			t.Errorf("%v, %d: expected %v, got %v", testCase.cond.ByStatusCode, testCase.resp.StatusCode, testCase.result, result)
		}
	}
}

func TestRetryByHeader(t *testing.T) {
	testCases := []struct {
		cond   *byHeader
		resp   *http.Response
		result bool
	}{
		{
			cond: &byHeader{
				RetryCondition_ByHeader: &config.RetryCondition_ByHeader{
					ByHeader: &config.RetryConditionHeader{
						Name:  "Grpc-Status",
						Value: "5",
					},
				},
			},
			resp: &http.Response{
				Header: http.Header{
					"Grpc-Status": []string{"5"},
				},
			},
			result: true,
		},
		{
			cond: &byHeader{
				RetryCondition_ByHeader: &config.RetryCondition_ByHeader{
					ByHeader: &config.RetryConditionHeader{
						Name:  "Grpc-Status",
						Value: `["5", "15"]`,
					},
				},
			},
			resp: &http.Response{
				Header: http.Header{
					"Grpc-Status": []string{"10"},
				},
			},
			result: false,
		},
		{
			cond: &byHeader{
				RetryCondition_ByHeader: &config.RetryCondition_ByHeader{
					ByHeader: &config.RetryConditionHeader{
						Name:  "Grpc-Status",
						Value: `["5","15"]`,
					},
				},
			},
			resp: &http.Response{
				Header: http.Header{
					"Grpc-Status": []string{"15"},
				},
			},
			result: true,
		},
		{
			cond: &byHeader{
				RetryCondition_ByHeader: &config.RetryCondition_ByHeader{
					ByHeader: &config.RetryConditionHeader{
						Name:  "Grpc-Status",
						Value: `16`,
					},
				},
			},
			resp: &http.Response{
				Header: http.Header{
					"Grpc-Status": []string{"16"},
				},
			},
			result: true,
		},
		{
			cond: &byHeader{
				RetryCondition_ByHeader: &config.RetryCondition_ByHeader{
					ByHeader: &config.RetryConditionHeader{
						Name:  "xxx-should-retry",
						Value: "true",
					},
				},
			},
			resp: &http.Response{
				Header: http.Header{
					"Xxx-Should-Retry": []string{"true"},
				},
			},
			result: true,
		},
		{
			cond: &byHeader{
				RetryCondition_ByHeader: &config.RetryCondition_ByHeader{
					ByHeader: &config.RetryConditionHeader{
						Name:  "xxx-should-retry",
						Value: "true",
					},
				},
			},
			resp: &http.Response{
				Header: http.Header{},
			},
			result: false,
		},
	}
	for _, testCase := range testCases {
		if err := testCase.cond.prepare(); err != nil {
			t.Errorf("prepare error: %v", err)
		}
		result := testCase.cond.judge(testCase.resp)
		if result != testCase.result {
			t.Errorf("%v, %v: expected %v, got %v", testCase.cond.ByHeader, testCase.resp.Header, testCase.result, result)
		}
	}
}

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
