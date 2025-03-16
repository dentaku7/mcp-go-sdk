package main

import (
	"encoding/json"

	"mcp-go-sdk/server"
	"mcp-go-sdk/transport"
)

// EchoTool implements a simple echo tool that returns the input message
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

	// Register the echo tool
	if err := srv.RegisterTool(&EchoTool{}); err != nil {
		panic(err)
	}

	// Start the server
	if err := srv.Start(); err != nil {
		panic(err)
	}
}
