package main

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
)

func TestRunFunction(t *testing.T) {

	type args struct {
		ctx context.Context
		req *fnv1.RunFunctionRequest
		ai  agentInvoker
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
		"ResponseIsReturned": {
			reason: "The Function should return a fatal result if credential cannot be found.",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "openai.fn.upbound.io/v1alpha1",
						"kind": "Prompt",
						"systemPrompt": "I'm a system",
						"userPrompt": "I'm a user"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_FATAL,
							Message:  `cannot get ANTHROPIC_API_KEY from credential "claude": claude: credential not found`,
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
				},
			},
		},
		"SimpleCompositionPipeline": {
			reason: "We should go through the composition pipeline without error.",
			args: args{
				ai: &mockAgentInvoker{
					InvokeFn: func(_ context.Context, _, _, _ string) (string, error) {
						return `---
apiVersion: some.group/v1
metadata:
  name: some-name
  annotations:
    upbound.io/name: some-name
`, nil
					},
				},
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
								"apiVersion": "openai.fn.upbound.io/v1alpha1",
								"kind": "Prompt",
								"systemPrompt": "I'm a system",
								"userPrompt": "I'm a user"
							}`),
					Credentials: mockCredentials(),
					Observed: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: &structpb.Struct{
								Fields: map[string]*structpb.Value{},
							},
						},
					},
					Desired: &fnv1.State{
						Composite: &fnv1.Resource{},
					},
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Desired: &fnv1.State{
						Composite: &fnv1.Resource{},
						Resources: map[string]*fnv1.Resource{
							"some-name": {
								Resource: &structpb.Struct{
									Fields: map[string]*structpb.Value{
										"apiVersion": {
											Kind: &structpb.Value_StringValue{
												StringValue: "some.group/v1",
											},
										},
										"metadata": {
											Kind: &structpb.Value_StructValue{
												StructValue: &structpb.Struct{
													Fields: map[string]*structpb.Value{
														"annotations": {
															Kind: &structpb.Value_StructValue{
																StructValue: &structpb.Struct{
																	Fields: map[string]*structpb.Value{
																		"upbound.io/name": {
																			Kind: &structpb.Value_StringValue{
																				StringValue: "some-name",
																			},
																		},
																	},
																},
															},
														},
														"name": {
															Kind: &structpb.Value_StringValue{
																StringValue: "some-name",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"SimpleOperationPipeline": {
			reason: "We should go through the operation pipeline without error.",
			args: args{
				ai: &mockAgentInvoker{
					InvokeFn: func(_ context.Context, _, _, _ string) (string, error) {
						return `some-response`, nil
					},
				},
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "openai.fn.upbound.io/v1alpha1",
						"kind": "Prompt",
						"systemPrompt": "I'm a system",
						"userPrompt": "I'm a user"
					}`),
					Credentials: mockCredentials(),
					RequiredResources: map[string]*fnv1.Resources{
						"ops.crossplane.io/watched-resource": {
							Items: []*fnv1.Resource{
								{
									Resource: &structpb.Struct{},
								},
							},
						},
					},
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{
						Tag: "hello",
						Ttl: &durationpb.Duration{
							Seconds: 60,
						},
					},
					Results: []*fnv1.Result{{
						Severity: fnv1.Severity_SEVERITY_NORMAL,
						Message:  "some-response",
						Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
					}},
					Conditions: []*fnv1.Condition{{
						Type:   "FunctionSuccess",
						Status: fnv1.Status_STATUS_CONDITION_TRUE,
						Reason: "Success",
						Target: fnv1.Target_TARGET_COMPOSITE_AND_CLAIM.Enum(),
					}},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := &Function{log: logging.NewNopLogger(), ai: tc.args.ai}
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

func mockCredentials() map[string]*fnv1.Credentials {
	return map[string]*fnv1.Credentials{
		credName: {
			Source: &fnv1.Credentials_CredentialData{
				CredentialData: &fnv1.CredentialData{
					Data: map[string][]byte{
						credKey: []byte("data"),
					},
				},
			},
		},
	}
}

type mockAgentInvoker struct {
	InvokeFn func(ctx context.Context, key, system, prompt string) (string, error)
}

func (m *mockAgentInvoker) Invoke(ctx context.Context, key, system, prompt string) (string, error) {
	return m.InvokeFn(ctx, key, system, prompt)
}
