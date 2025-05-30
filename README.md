# MCP-Go SDK

A Go SDK for building Model Communication Protocol (MCP) tools and servers. This SDK provides the building blocks for implementing MCP-compliant tools that can be used with AI applications like Cursor IDE.

## Quick Start

Check out the example server in `servers/example` to see a minimal implementation of an MCP tool that echoes back messages:

```go
package main

import (
    "encoding/json"
    "mcp-go/server"
    "mcp-go/transport"
)

// EchoTool implements a simple echo tool
type EchoTool struct{}

func (t *EchoTool) Name() string {
    return "echo"
}

func (t *EchoTool) Description() string {
    return "A simple echo tool that returns the input message"
}

func (t *EchoTool) Schema() json.RawMessage {
    return json.RawMessage(`{
        "type": "object",
        "properties": {
            "message": {
                "type": "string",
                "description": "The message to echo back"
            }
        },
        "required": ["message"]
    }`)
}

func (t *EchoTool) Execute(params json.RawMessage) (interface{}, error) {
    var input struct {
        Message string `json:"message"`
    }
    if err := json.Unmarshal(params, &input); err != nil {
        return nil, err
    }
    
    return map[string]interface{}{
        "content": []map[string]interface{}{
            {
                "type": "text",
                "text": input.Message,
            },
        },
        "metadata": map[string]interface{}{
            "length": len(input.Message),
        },
    }, nil
}

func main() {
    // Create a new server with stdin/stdout transport
    srv := server.NewServer(transport.NewStdioTransport())
    
    // Register your tool
    if err := srv.RegisterTool(&EchoTool{}); err != nil {
        panic(err)
    }
    
    // Start the server
    if err := srv.Start(); err != nil {
        panic(err)
    }
}

## Core Concepts

### 1. Tools

A Tool in MCP is a service that can be called by AI applications. Each tool must implement the `mcp.Tool` interface:

```go
type Tool interface {
    // Name returns the unique identifier for this tool
    Name() string
    
    // Description returns a human-readable description
    Description() string
    
    // Schema returns the JSON schema for the tool's parameters
    Schema() json.RawMessage
    
    // Execute runs the tool with the given parameters
    Execute(params json.RawMessage) (interface{}, error)
}
```

### 2. Response Format

Tools should return responses in the MCP format:

```go
{
    "content": [
        {
            "type": "text",
            "text": "Your response text"
        }
    ],
    "metadata": {
        // Optional metadata about the response
    }
}
```

### 3. Transport Layer

The SDK provides a flexible transport layer through the `Transport` interface:

```go
type Transport interface {
    Send(data interface{}) error
    Receive() ([]byte, error)
    Close() error
}
```

By default, the SDK includes a stdio transport (`transport.NewStdioTransport()`) for command-line tools.

### 4. Configuration

To use your MCP tool with Cursor IDE, create a `.cursor/mcp.json` in your project root:

```json
{
    "mcpServers": {
        "mytool": {
            "command": "mytool",
            "args": [],
            "env": {}
        }
    }
}
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

Please ensure your code:
- Follows Go best practices
- Includes appropriate documentation
- Has test coverage
- Handles errors appropriately
