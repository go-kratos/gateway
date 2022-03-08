package condition

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
				Condition_ByStatusCode: &config.Condition_ByStatusCode{
					ByStatusCode: "501",
				},
			},
			resp:   &http.Response{StatusCode: 501},
			result: true,
		},
		{
			cond: &byStatusCode{
				Condition_ByStatusCode: &config.Condition_ByStatusCode{
					ByStatusCode: "501-509",
				},
			},
			resp:   &http.Response{StatusCode: 500},
			result: false,
		},
		{
			cond: &byStatusCode{
				Condition_ByStatusCode: &config.Condition_ByStatusCode{
					ByStatusCode: "501-509",
				},
			},
			resp:   &http.Response{StatusCode: 502},
			result: true,
		},
	}

	for _, testCase := range testCases {
		if err := testCase.cond.Prepare(); err != nil {
			t.Errorf("prepare error: %v", err)
		}
		result := testCase.cond.Judge(testCase.resp)
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
				Condition_ByHeader: &config.Condition_ByHeader{
					ByHeader: &config.ConditionHeader{
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
				Condition_ByHeader: &config.Condition_ByHeader{
					ByHeader: &config.ConditionHeader{
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
				Condition_ByHeader: &config.Condition_ByHeader{
					ByHeader: &config.ConditionHeader{
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
				Condition_ByHeader: &config.Condition_ByHeader{
					ByHeader: &config.ConditionHeader{
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
				Condition_ByHeader: &config.Condition_ByHeader{
					ByHeader: &config.ConditionHeader{
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
				Condition_ByHeader: &config.Condition_ByHeader{
					ByHeader: &config.ConditionHeader{
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
		if err := testCase.cond.Prepare(); err != nil {
			t.Errorf("prepare error: %v", err)
		}
		result := testCase.cond.Judge(testCase.resp)
		if result != testCase.result {
			t.Errorf("%v, %v: expected %v, got %v", testCase.cond.ByHeader, testCase.resp.Header, testCase.result, result)
		}
	}
}
