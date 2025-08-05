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

import "github.com/crossplane/function-sdk-go/errors"

// Config represents an MCP toplevel configuration.
type Config struct {
	Transport Transport `json:"transport"`
	BaseURL   string    `json:"baseURL"`
}

// Transport defines specific transport types that are supported.
type Transport string

var (
	// SSE represents Server-Sent Events.
	SSE Transport = "sse"
	// StreamableHTTP represents Streamable HTTP.
	StreamableHTTP Transport = "http-stream"
)

// Valid returns no error if the provided Config is valid.
func (c Config) Valid() error {
	if len(c.BaseURL) == 0 {
		return errors.New("invalid mcp config: baseURL required")
	}

	switch c.Transport {
	case SSE, StreamableHTTP:
		return nil
	default:
		return errors.New("invalid mcp config: transport must be one of 'sse' or 'http-stream")
	}
}
