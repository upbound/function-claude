package tool

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMerge(t *testing.T) {
	type args struct {
		current Config
		new     Config
	}
	type want struct {
		res Config
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"CurrentTransportUnset": {
			reason: "If new has a transport, then current's transport should be set.",
			args: args{
				current: Config{
					Transport: "",
				},
				new: Config{
					Transport: "sse",
				},
			},
			want: want{
				res: Config{
					Transport: "sse",
				},
			},
		},
		"CurrentBaseURLUnset": {
			reason: "If new has a transport, then current's transport should be set.",
			args: args{
				current: Config{
					BaseURL: "",
				},
				new: Config{
					BaseURL: "http://local",
				},
			},
			want: want{
				res: Config{
					BaseURL: "http://local",
				},
			},
		},
		"CurrentTransportSet": {
			reason: "If transport is currently set, then nothing to do.",
			args: args{
				current: Config{
					Transport: "some-thing",
				},
				new: Config{
					Transport: "sse",
				},
			},
			want: want{
				res: Config{
					Transport: "some-thing",
				},
			},
		},
		"CurrentBaseURLSet": {
			reason: "If baseURL is currently set, then nothing to do.",
			args: args{
				current: Config{
					BaseURL: "some-thing",
				},
				new: Config{
					BaseURL: "http://local",
				},
			},
			want: want{
				res: Config{
					BaseURL: "some-thing",
				},
			},
		},
		"PartialTransportSet": {
			reason: "If new has a transport, then current's transport should be set.",
			args: args{
				current: Config{
					Transport: "some-thing",
				},
				new: Config{
					BaseURL: "http://local",
				},
			},
			want: want{
				res: Config{
					Transport: "some-thing",
					BaseURL:   "http://local",
				},
			},
		},
		"PartialBaseURLSet": {
			reason: "If new has a transport, then current's transport should be set.",
			args: args{
				current: Config{
					BaseURL: "some-thing",
				},
				new: Config{
					Transport: "sse",
				},
			},
			want: want{
				res: Config{
					Transport: "sse",
					BaseURL:   "some-thing",
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := NewResolver().merge(tc.args.current, tc.args.new)

			if diff := cmp.Diff(tc.want.res, got); diff != "" {
				t.Errorf("\n%s\nMerge(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestParse(t *testing.T) {
	type args struct {
		e string
	}
	type want struct {
		key string
		res Config
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"MCP_SERVER_TOOL_*_TRANSPORT=sse": {
			args: args{
				e: "MCP_SERVER_TOOL_K1_TRANSPORT=sse",
			},
			want: want{
				key: "k1",
				res: Config{
					Transport: SSE,
				},
			},
		},
		"MCP_SERVER_TOOL_*_BASEURL=http://local": {
			args: args{
				e: "MCP_SERVER_TOOL_K1_BASEURL=http://local",
			},
			want: want{
				key: "k1",
				res: Config{
					BaseURL: "http://local",
				},
			},
		},
		"MCP_SERVER_TOOL_*_TRANSPOR=sse": {
			args: args{
				e: "MCP_SERVER_TOOL_K1_TRANSPOR=sse",
			},
			want: want{
				key: "k1",
				res: Config{},
			},
		},
		"MCP_SERVER_TOOL_*_BASEU=http://local": {
			args: args{
				e: "MCP_SERVER_TOOL_K1_BASEU=http://local",
			},
			want: want{
				key: "k1",
				res: Config{},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			key, got := NewResolver().parse(tc.args.e)

			if diff := cmp.Diff(tc.want.key, key); diff != "" {
				t.Errorf("\n\nSplit(...): -want, +got:\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.res, got); diff != "" {
				t.Errorf("\n\nSplit(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestFromEnvVars(t *testing.T) {
	type args struct {
		eg environGetter
	}
	type want struct {
		res map[string]Config
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"NoEnvVars": {
			args: args{
				eg: &mockEnvironGetter{
					EnvironFn: func() []string {
						return []string{}
					},
				},
			},
			want: want{
				res: map[string]Config{},
			},
		},
		"HasMCP_SERVER_TOOL_K1_*": {
			args: args{
				eg: &mockEnvironGetter{
					EnvironFn: func() []string {
						return []string{
							"MCP_SERVER_TOOL_K1_TRANSPORT=sse",
							"MCP_SERVER_TOOL_K1_BASEURL=http://local",
						}
					},
				},
			},
			want: want{
				res: map[string]Config{
					"k1": {
						Transport: SSE,
						BaseURL:   "http://local",
					},
				},
			},
		},
		"HasMiscWithMCP_SERVER_TOOL_*": {
			args: args{
				eg: &mockEnvironGetter{
					EnvironFn: func() []string {
						return []string{
							"SOME=value",
							"MCP_SERVER_TOOL_K1_TRANSPORT=sse",
							"MCP_SERVER_TOOL_K1_BASEURL=http://local",
						}
					},
				},
			},
			want: want{
				res: map[string]Config{
					"k1": {
						Transport: SSE,
						BaseURL:   "http://local",
					},
				},
			},
		},
		"HasInvalidMCP_SERVER_TOOL_*": {
			args: args{
				eg: &mockEnvironGetter{
					EnvironFn: func() []string {
						return []string{
							"SOME=value",
							"MCP_SERVER_TOOL_K1_TRANSPOR=sse",
							"MCP_SERVER_TOOL_K1_BASEURL=http://local",
						}
					},
				},
			},
			want: want{
				res: map[string]Config{},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			r := NewResolver(withEnvironGetter(tc.args.eg))
			got := r.FromEnvVars()

			if diff := cmp.Diff(tc.want.res, got); diff != "" {
				t.Errorf("\n\nFromEnvVars(...): -want, +got:\n%s", diff)
			}
		})
	}
}

// Helper mocks for working against the Environ calls.
func withEnvironGetter(eg environGetter) Option {
	return func(r *Resolver) {
		r.eg = eg
	}
}

type mockEnvironGetter struct {
	EnvironFn func() []string
}

func (m *mockEnvironGetter) Environ() []string {
	return m.EnvironFn()
}
