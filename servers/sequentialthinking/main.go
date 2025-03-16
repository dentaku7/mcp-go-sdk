package main

import (
	"log"

	"mcp-go-sdk/server"
	"mcp-go-sdk/transport"
)

func main() {
	// Create a new MCP server with stdio transport
	srv := server.NewServer(transport.NewStdioTransport())

	// Create and register the sequential thinking tool
	tool := NewTool()
	if err := srv.RegisterTool(tool); err != nil {
		log.Fatalf("Failed to register tool: %v", err)
	}

	// Start the server
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
