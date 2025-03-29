package mcp

import (
	"encoding/json"
)

// Transport defines the interface for MCP communication
type Transport interface {
	// Send sends data through the transport
	Send(data interface{}) error

	// Receive reads data from the transport
	Receive() ([]byte, error)

	// Close implements proper resource cleanup
	Close() error
}

// TransportConfig represents configuration options for a transport
type TransportConfig struct {
	// BufferSize is the size of the read/write buffers
	BufferSize int
}

// DefaultTransportConfig returns the default transport configuration
func DefaultTransportConfig() *TransportConfig {
	return &TransportConfig{
		BufferSize: 4096,
	}
}

// Tool represents a tool that can be executed by the MCP server
type Tool interface {
	// Name returns the name of the tool
	Name() string

	// Description returns a description of what the tool does
	Description() string

	// Schema returns the JSON schema for the tool's arguments
	Schema() json.RawMessage

	// Execute runs the tool with the given arguments
	Execute(params json.RawMessage) (interface{}, error)
}

// Request represents a JSON-RPC request
type Request struct {
	JsonRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC response
type Response struct {
	JsonRPC string          `json:"jsonrpc"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	ID      json.RawMessage `json:"id"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Notification represents a JSON-RPC notification
type Notification struct {
	JsonRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// InitializeRequest represents an initialize request
type InitializeRequest struct {
	JsonRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  struct {
		ProtocolVersion string `json:"protocolVersion"`
	} `json:"params"`
}

// InitializeResult represents the result of an initialize request
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// ServerCapabilities represents the server's capabilities
type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools"`
}

// ToolsCapability represents the server's tool capabilities
type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// ServerInfo represents information about the server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ListToolsResponse represents the response to a tools/list request
type ListToolsResponse struct {
	Tools      []ToolInfo `json:"tools"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

// ToolInfo represents information about a tool
type ToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// CallToolRequest represents a tool call request
type CallToolRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Meta      *CallToolMeta   `json:"_meta,omitempty"`
}

// CallToolMeta represents metadata for a tool call
type CallToolMeta struct {
	ProgressToken int `json:"progressToken,omitempty"`
}

// ToolResponse represents a successful tool execution response
type ToolResponse struct {
	Content []ToolContent `json:"content"`
}

// ToolError represents a tool execution error response
type ToolError struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError"`
}

// ToolContent represents a piece of content in a tool response
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
