package server

// Protocol versions
const (
	Version         = "2.0"        // JSON-RPC version
	ProtocolVersion = "2024-11-05" // MCP protocol version
)

// Method names
const (
	MethodInitialize  = "initialize"
	MethodInitialized = "initialized"
	MethodListTools   = "tools/list"
	MethodCallTool    = "tools/call"
)

// Error codes as per JSON-RPC 2.0 specification
const (
	ErrParseError     = -32700 // Invalid JSON
	ErrInvalidRequest = -32600 // The JSON sent is not a valid Request object
	ErrMethodNotFound = -32601 // The method does not exist / is not available
	ErrInvalidParams  = -32602 // Invalid method parameter(s)
	ErrInternal       = -32603 // Internal JSON-RPC error
)
