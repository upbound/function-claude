package tool

import "github.com/crossplane/function-sdk-go/errors"

type Config struct {
	Transport Transport `json:"transport"`
	BaseURL   string    `json:"baseURL"`
}

type Transport string

var (
	SSE            Transport = "sse"
	StreamableHTTP Transport = "http-stream"
)

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
