/*
Copyright 2025 The Upbound Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
		"IgnoredResource": {
			reason: "We should return early if the incoming resource should be ignored.",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Context: &structpb.Struct{Fields: map[string]*structpb.Value{
						"ops.upbound.io/ignored-resource": structpb.NewBoolValue(true),
					}},
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Context: &structpb.Struct{Fields: map[string]*structpb.Value{
						"ops.upbound.io/ignored-resource": structpb.NewBoolValue(true),
					}},
					Meta: &fnv1.ResponseMeta{
						Ttl: &durationpb.Duration{
							Seconds: 60,
						},
					},
					Conditions: []*fnv1.Condition{
						{
							Type:   "FunctionSuccess",
							Status: fnv1.Status_STATUS_CONDITION_TRUE,
							Reason: "Success",
							Target: fnv1.Target_TARGET_COMPOSITE_AND_CLAIM.Enum(),
						},
					},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_NORMAL,
							Message:  "received an ignored resource, skipping",
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
				},
			},
		},
		"ResponseIsReturned": {
			reason: "The Function should return a fatal result if credential cannot be found.",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "claude.fn.upbound.io/v1alpha1",
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
					InvokeFn: func(_ context.Context, _, _, _, _ string) (string, error) {
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
								"apiVersion": "claude.fn.upbound.io/v1alpha1",
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
					InvokeFn: func(_ context.Context, _, _, _, _ string) (string, error) {
						return `some-response`, nil
					},
				},
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "claude.fn.upbound.io/v1alpha1",
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
					Desired: &fnv1.State{},
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
					Desired: &fnv1.State{},
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

func TestResourceFrom(t *testing.T) {
	type args struct {
		resp string
	}
	type want struct {
		resource map[string]*fnv1.Resource
		err      error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"String": {
			reason: "We should return an error if we received a string that is neither JSON nor YAML",
			args: args{
				resp: "some-response",
			},
			want: want{
				err: cmpopts.AnyError,
			},
		},
		"ValidJSON": {
			reason: "We should not return an error if we processed valid JSON",
			args: args{
				resp: `{}`,
			},
			want: want{
				resource: map[string]*fnv1.Resource{
					"": {Resource: &structpb.Struct{}},
				},
			},
		},
		"ValidYAML": {
			reason: "We should not return an error if we processed valid YAML",
			args: args{
				resp: `a: b`,
			},
			want: want{
				resource: map[string]*fnv1.Resource{
					"": {Resource: &structpb.Struct{Fields: map[string]*structpb.Value{"a": structpb.NewStringValue("b")}}},
				},
			},
		},
		"InvalidJSON": {
			reason: "We should return an error if we attempt to process invalid JSON",
			args: args{
				resp: `{a: `,
			},
			want: want{
				err: cmpopts.AnyError,
			},
		},
		"InvalidYAML": {
			reason: "We should return an error if we attempt to process invalid YAML",
			args: args{
				resp: ``,
			},
			want: want{
				err: cmpopts.AnyError,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := &Function{log: logging.NewNopLogger()}
			got, err := f.resourceFrom(tc.args.resp)

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.resource, got, protocmp.Transform()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s", tc.reason, diff)
			}
		})
	}
}

type mockAgentInvoker struct {
	InvokeFn func(ctx context.Context, key, system, prompt, modelName string) (string, error)
}

func (m *mockAgentInvoker) Invoke(ctx context.Context, key, system, prompt, modelName string) (string, error) {
	return m.InvokeFn(ctx, key, system, prompt, modelName)
}
