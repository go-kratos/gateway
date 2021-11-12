package client

import (
	"net/http"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
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
					ByStatusCode: &config.RetryConditionByStatusCode{
						StatusCodes: []string{"501"},
					},
				},
			},
			resp:   &http.Response{StatusCode: 501},
			result: true,
		},
		{
			cond: &byStatusCode{
				RetryCondition_ByStatusCode: &config.RetryCondition_ByStatusCode{
					ByStatusCode: &config.RetryConditionByStatusCode{
						StatusCodes: []string{"501-509"},
					},
				},
			},
			resp:   &http.Response{StatusCode: 500},
			result: false,
		},
		{
			cond: &byStatusCode{
				RetryCondition_ByStatusCode: &config.RetryCondition_ByStatusCode{
					ByStatusCode: &config.RetryConditionByStatusCode{
						StatusCodes: []string{"501-509"},
					},
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
			t.Errorf("%v, %d: expected %v, got %v", testCase.cond.ByStatusCode.StatusCodes, testCase.resp.StatusCode, testCase.result, result)
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
					ByHeader: &config.RetryConditionByHeader{
						Headers: []*config.RetryConditionHeader{
							{
								Name:   "Grpc-Status",
								Values: []string{`"NOT_FOUND"`},
							},
						},
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
					ByHeader: &config.RetryConditionByHeader{
						Headers: []*config.RetryConditionHeader{
							{
								Name:   "Grpc-Status",
								Values: []string{`"NOT_FOUND"`, `"DATA_LOSS"`},
							},
						},
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
					ByHeader: &config.RetryConditionByHeader{
						Headers: []*config.RetryConditionHeader{
							{
								Name:   "Grpc-Status",
								Values: []string{`"NOT_FOUND"`, `"DATA_LOSS"`},
							},
						},
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
					ByHeader: &config.RetryConditionByHeader{
						Headers: []*config.RetryConditionHeader{
							{
								Name:   "Grpc-Status",
								Values: []string{`16`},
							},
						},
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
					ByHeader: &config.RetryConditionByHeader{
						Headers: []*config.RetryConditionHeader{
							{
								Name:   "xxx-should-retry",
								Values: []string{"true"},
							},
						},
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
					ByHeader: &config.RetryConditionByHeader{
						Headers: []*config.RetryConditionHeader{
							{
								Name:   "xxx-should-retry",
								Values: []string{"true"},
							},
						},
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
			t.Errorf("%v, %v: expected %v, got %v", testCase.cond.ByHeader.Headers, testCase.resp.Header, testCase.result, result)
		}
	}
}
