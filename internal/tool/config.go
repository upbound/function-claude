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
