// /*
// Copyright 2025 The Upbound Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package tool

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestValidate(t *testing.T) {
	type args struct {
		config Config
	}
	type want struct {
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"InvalidIncorrectTransport": {
			reason: "If an invalid transport is supplied, validation should fail.",
			args: args{
				config: Config{
					Transport: "stdio",
					BaseURL:   "./local",
				},
			},
			want: want{
				err: cmpopts.AnyError,
			},
		},
		"InvalidBadBaseURL": {
			reason: "If an invalid baseURL is supplied, validation should fail.",
			args: args{
				config: Config{
					Transport: "http-stream",
					BaseURL:   "",
				},
			},
			want: want{
				err: cmpopts.AnyError,
			},
		},
		"ValidConfigSSE": {
			reason: "If a valid SSE config is supplied, no error should be returned.",
			args: args{
				config: Config{
					Transport: "sse",
					BaseURL:   "http://localhost/sse",
				},
			},
		},
		"ValidConfigStreamableHTTP": {
			reason: "If a valid StreamableHTTP config is supplied, no error should be returned.",
			args: args{
				config: Config{
					Transport: "http-stream",
					BaseURL:   "http://localhost/mcp",
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.args.config.Valid()

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nValid(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}
