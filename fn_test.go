package main

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

func TestRunFunction(t *testing.T) {
	type args struct {
		ctx context.Context
		req *fnv1.RunFunctionRequest
	}
	type want struct {
		rsp *fnv1.RunFunctionResponse
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		// TODO(negz): Add some tests. :D
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := NewFunction(logging.NewNopLogger())
			rsp, err := f.RunFunction(tc.args.ctx, tc.args.req)

			if diff := cmp.Diff(tc.want.rsp, rsp, protocmp.Transform()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestContextToYAML(t *testing.T) {
	// Create test context similar to what function-rds-metrics produces
	metricsData := map[string]interface{}{
		"databaseName": "test-db",
		"region":       "us-west-2",
		"timestamp":    "2025-07-28T21:27:38+02:00",
		"metrics": map[string]interface{}{
			"CPUUtilization": map[string]interface{}{
				"value":     75.5,
				"unit":      "Percent",
				"timestamp": "2025-07-28T21:27:38+02:00",
			},
		},
	}

	contextData := map[string]interface{}{
		"metricsResult": metricsData,
		"rdsMetricsRef": "metricsResult",
		"otherData":     "should not be included",
	}

	ctx, err := structpb.NewStruct(contextData)
	if err != nil {
		t.Fatalf("Failed to create test context: %v", err)
	}

	tests := []struct {
		name         string
		contextFields []string
		expectEmpty  bool
		shouldContain []string
		shouldNotContain []string
	}{
		{
			name:         "Extract metricsResult",
			contextFields: []string{"metricsResult"},
			expectEmpty:  false,
			shouldContain: []string{"databaseName: test-db", "CPUUtilization", "region: us-west-2"},
			shouldNotContain: []string{"otherData", "rdsMetricsRef"},
		},
		{
			name:         "Extract multiple fields",
			contextFields: []string{"metricsResult", "rdsMetricsRef"},
			expectEmpty:  false,
			shouldContain: []string{"metricsResult:", "rdsMetricsRef: metricsResult"},
			shouldNotContain: []string{"otherData"},
		},
		{
			name:         "No fields specified",
			contextFields: []string{},
			expectEmpty:  true,
		},
		{
			name:         "Non-existent field",
			contextFields: []string{"nonExistent"},
			expectEmpty:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml, err := ContextToYAML(ctx, tt.contextFields)
			if err != nil {
				t.Fatalf("ContextToYAML failed: %v", err)
			}

			if tt.expectEmpty {
				if yaml != "" {
					t.Errorf("Expected empty YAML, got: %s", yaml)
				}
				return
			}

			if yaml == "" {
				t.Fatal("Expected YAML output, got empty string")
			}

			t.Logf("Generated YAML for %s:\n%s", tt.name, yaml)

			// Check expected content
			for _, expected := range tt.shouldContain {
				if !strings.Contains(yaml, expected) {
					t.Errorf("YAML should contain '%s'", expected)
				}
			}

			// Check unexpected content
			for _, unexpected := range tt.shouldNotContain {
				if strings.Contains(yaml, unexpected) {
					t.Errorf("YAML should not contain '%s'", unexpected)
				}
			}
		})
	}
}

func TestContextToYAMLEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *structpb.Struct
		fields  []string
		expectEmpty bool
	}{
		{
			name:    "Nil context",
			ctx:     nil,
			fields:  []string{"field"},
			expectEmpty: true,
		},
		{
			name:    "Empty context",
			ctx:     &structpb.Struct{Fields: make(map[string]*structpb.Value)},
			fields:  []string{"field"},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml, err := ContextToYAML(tt.ctx, tt.fields)
			if err != nil {
				t.Fatalf("ContextToYAML failed: %v", err)
			}

			if tt.expectEmpty && yaml != "" {
				t.Errorf("Expected empty YAML, got: %s", yaml)
			}
		})
	}
}
