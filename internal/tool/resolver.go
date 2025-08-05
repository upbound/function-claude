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
	"context"
	"os"
	"regexp"
	"strings"

	mcpadapter "github.com/i2y/langchaingo-mcp-adapter"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/tmc/langchaingo/tools"

	"github.com/crossplane/function-sdk-go/logging"
)

var (
	re                   = regexp.MustCompile(`MCP_SERVER_TOOL_(?P<key>.*)_(?P<type>.*)`)
	defaultEnvironGetter = &osEnvironGetter{}
)

const (
	key     = "key"
	cfgtype = "type"
)

// Resolver is used for resolving MCP server configs from the environment
// and converting them into langchaingo tools.
type Resolver struct {
	log logging.Logger
	eg  environGetter
}

// Option modifies the underlying Resolver.
type Option func(*Resolver)

// WithLogger overrides the underlying logger for the Resolver.
func WithLogger(log logging.Logger) Option {
	return func(r *Resolver) {
		r.log = log
	}
}

// NewResolver constructs a Resolver.
func NewResolver(opts ...Option) *Resolver {
	r := &Resolver{
		log: logging.NewNopLogger(),
		eg:  defaultEnvironGetter,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Resolve resolves the tools available from the supplied MCP server
// configurations. If errors occur along the way, the errors are logged
// and the tools for those servers are not returned.
func (r *Resolver) Resolve(ctx context.Context, cfgs map[string]Config) []tools.Tool {
	res := make([]tools.Tool, 0)
	for _, v := range cfgs {

		var mc *mcpclient.Client
		var err error

		switch v.Transport {
		case SSE:
			mc, err = mcpclient.NewSSEMCPClient(v.BaseURL)
		case StreamableHTTP:
			mc, err = mcpclient.NewStreamableHttpClient(v.BaseURL)
		}

		log := r.log.WithValues("transport", v.Transport, "baseURL", v.BaseURL)

		if err != nil {
			log.Info("failed to initialize mcp client for server", "error", err)
			continue
		}

		// Start the client
		if err := mc.Start(ctx); err != nil {
			log.Info("failed to start mcp client", "error", err)
			continue
		}
		log.Debug("mcp client successfully started")

		// Create the adapter for this server
		adapter, err := mcpadapter.New(mc)
		if err != nil {
			log.Info("failed to initialize langchain adapter for mcp server", "error", err)
			continue
		}

		// Get tools from this MCP server
		tools, err := adapter.Tools()
		if err != nil {
			log.Info("failed to get the available tools from mcp server", "error", err)
			continue
		}

		// Aggregate tools from this server
		res = append(res, tools...)
		log.Debug("successfully added tools from mcp server", "tools", toolString(tools))
	}

	return res
}

// FromEnvVars derives Configs for MCP servers from the environment variables
// supplied to the process. If the resulting Config is invalid, it is not
// returned.
func (r *Resolver) FromEnvVars() map[string]Config {
	cfgs := map[string]Config{}

	for _, e := range r.eg.Environ() {
		if !strings.HasPrefix(e, "MCP_SERVER_TOOL_") {
			continue // not an env var that we're interested in.
		}
		k, newc := r.parse(e)
		current := cfgs[k]
		cfgs[k] = r.merge(current, newc)
	}

	// validate configs before setting as tools
	for k, v := range cfgs {
		if err := v.Valid(); err != nil {
			r.log.Info("invalid config, skipping", "error", err)
			delete(cfgs, k)
		}
	}

	return cfgs
}

// parse the supplied k=v environment variable from an MCP_SERVER_TOOL_*
// environment variable.
func (r *Resolver) parse(e string) (string, Config) {
	matches := re.FindStringSubmatch(e)

	names := re.SubexpNames()
	result := make(map[string]string)
	for i, name := range names {
		if i != 0 && name != "" { // Skip the full match and unnamed groups
			result[name] = strings.ToLower(matches[i])
		}
	}

	cfg := Config{}
	vtype := strings.Split(result[cfgtype], "=")
	switch vtype[0] {
	case "transport":
		cfg.Transport = Transport(vtype[1])
	case "baseurl":
		cfg.BaseURL = vtype[1]
	}

	return result[key], cfg
}

// merge two MCP server Configs. If the current Config has an unset value, the
// value from new is applied.
func (r *Resolver) merge(currentc, newc Config) Config {
	if currentc.Transport == "" && newc.Transport != "" {
		currentc.Transport = newc.Transport
	}
	if currentc.BaseURL == "" && newc.BaseURL != "" {
		currentc.BaseURL = newc.BaseURL
	}

	return currentc
}

type environGetter interface {
	Environ() []string
}

type osEnvironGetter struct{}

// Environ returns the result of retrieving the environment variables from the
// underlying OS.
func (o *osEnvironGetter) Environ() []string {
	return os.Environ()
}

// toolString helps with printing out tools.Tool names as they don't natively
// implement String().
func toolString(tools []tools.Tool) []string {
	res := []string{}
	for _, t := range tools {
		res = append(res, t.Name())
	}
	return res
}
