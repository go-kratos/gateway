package client

import (
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
)

func TestParseRetryConditon(t *testing.T) {
	testCases := []struct {
		endpoint   *config.Endpoint
		conditions [][]uint32
	}{
		{
			endpoint: &config.Endpoint{
				Retry: &config.Retry{
					Conditions: []string{"500"},
				},
			},
			conditions: [][]uint32{{500}},
		},
		{
			endpoint: &config.Endpoint{
				Retry: &config.Retry{
					Conditions: []string{"501", "502"},
				},
			},
			conditions: [][]uint32{{501}, {502}},
		},
		{
			endpoint: &config.Endpoint{
				Retry: &config.Retry{
					Conditions: []string{"400-500", "501"},
				},
			},
			conditions: [][]uint32{{400, 500}, {501}},
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
