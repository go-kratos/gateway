package client

import (
	"net/http"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
)

func TestParseStatusCode(t *testing.T) {
	testCases := []struct {
		protocol   config.Protocol
		response   *http.Response
		statusCode uint32
	}{
		{
			protocol:   config.Protocol_HTTP,
			response:   &http.Response{StatusCode: 200},
			statusCode: 200,
		},
		{
			protocol:   config.Protocol_GRPC,
			response:   &http.Response{StatusCode: 0, Header: http.Header{"Grpc-Status": []string{"201"}}},
			statusCode: 201,
		},
	}

	for _, testCase := range testCases {
		statusCode := parseStatusCode(testCase.response, testCase.protocol)
		if statusCode != testCase.statusCode {
			t.Errorf("parseStatusCode(%v, %s) = %d, want %d", testCase.response, testCase.protocol, statusCode, testCase.statusCode)
		}
	}
}

func TestJudgeRetryRequired(t *testing.T) {
	testCases := []struct {
		conditions [][]uint32
		statusCode uint32
		retry      bool
	}{
		{
			conditions: [][]uint32{{500}, {501}, {502}},
			statusCode: 502,
			retry:      true,
		},
		{
			conditions: [][]uint32{{400, 500}},
			statusCode: 500,
			retry:      true,
		},
		{
			conditions: [][]uint32{{400, 500}},
			statusCode: 501,
			retry:      false,
		},
	}

	for _, testCase := range testCases {
		retry := judgeRetryRequired(testCase.conditions, testCase.statusCode)
		if retry != testCase.retry {
			t.Errorf("judgeRetryRequired(%v, %d) = %v, want %v", testCase.conditions, testCase.statusCode, retry, testCase.retry)
		}
	}
}
