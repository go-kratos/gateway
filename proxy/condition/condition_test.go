package condition

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
)

var nopBody = io.NopCloser(&bytes.Buffer{})

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
			resp:   &http.Response{StatusCode: 501, Body: nopBody},
			result: true,
		},
		{
			cond: &byStatusCode{
				Condition_ByStatusCode: &config.Condition_ByStatusCode{
					ByStatusCode: "501-509",
				},
			},
			resp:   &http.Response{StatusCode: 500, Body: nopBody},
			result: false,
		},
		{
			cond: &byStatusCode{
				Condition_ByStatusCode: &config.Condition_ByStatusCode{
					ByStatusCode: "501-509",
				},
			},
			resp:   &http.Response{StatusCode: 502, Body: nopBody},
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
				Body: nopBody,
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
				Body: nopBody,
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
				Body: nopBody,
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
				Body: nopBody,
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
				Body: nopBody,
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
				Body:   nopBody,
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
