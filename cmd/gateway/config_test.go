package main

import (
	"testing"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	loggingv1 "github.com/go-kratos/gateway/api/gateway/middleware/logging/v1"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

func equalTo() *configv1.Gateway {
	return &configv1.Gateway{
		Name: "helloworld",
		Hosts: []string{
			"localhost",
			"127.0.0.1",
		},
		Endpoints: []*configv1.Endpoint{
			{
				Path:     "/helloworld/*",
				Protocol: configv1.Protocol_HTTP,
				Timeout:  &durationpb.Duration{Seconds: 1},
				Backends: []*configv1.Backend{
					{
						Target: "127.0.0.1:8000",
					},
				},
			},
			{
				Path:     "/helloworld.Greeter/*",
				Method:   "POST",
				Protocol: configv1.Protocol_GRPC,
				Timeout:  &durationpb.Duration{Seconds: 1},
				Backends: []*configv1.Backend{
					{
						Target: "127.0.0.1:9000",
					},
				},
				Retry: &configv1.Retry{
					Attempts:      3,
					PerTryTimeout: &durationpb.Duration{Nanos: 500000000},
					Conditions: []*configv1.RetryCondition{
						{Condition: &configv1.RetryCondition_ByStatusCode{ByStatusCode: "502-504"}},
						{Condition: &configv1.RetryCondition_ByHeader{ByHeader: &configv1.RetryConditionHeader{
							Name:  "Grpc-Status",
							Value: "14",
						}}},
					},
				},
			},
		},
		Middlewares: []*configv1.Middleware{
			{
				Name:    "logging",
				Options: asAny(&loggingv1.Logging{}),
			},
		},
	}
}

func asAny(in proto.Message) *anypb.Any {
	out, err := anypb.New(in)
	if err != nil {
		panic(err)
	}
	return out
}

func TestConfigUnmarshaler(t *testing.T) {
	cfg := config.New(
		config.WithSource(
			file.NewSource("config.yaml"),
		),
	)
	if err := cfg.Load(); err != nil {
		t.Fatal(err)
	}
	gateway := &configv1.Gateway{}
	if err := cfg.Scan(gateway); err != nil {
		t.Fatal(err)
	}

	left, err := protojson.Marshal(gateway)
	if err != nil {
		t.Fatal(err)
	}
	right, err := protojson.Marshal(equalTo())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("gateway config:\nloaded: %s\nshould equal to: %s\n", left, right)

	if !proto.Equal(gateway, equalTo()) {
		t.Errorf("inconsistent gateway config")
	}
}
