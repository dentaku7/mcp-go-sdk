package transport

import (
	"os"

	"github.com/dentaku7/mcp-go-sdk"
)

// StdioTransport implements Transport using stdin/stdout
type StdioTransport struct {
	*BaseTransport
}

// NewStdioTransport creates a new transport that uses stdin/stdout
func NewStdioTransport() *BaseTransport {
	config := &mcp.TransportConfig{
		BufferSize: 4096,
	}
	return NewBaseTransport(os.Stdin, os.Stdout, nil, config)
}
