package main

import (
	"log"
	"os"
	"strings"

	"mcp-go-sdk/server"
	"mcp-go-sdk/transport"
)

func main() {
	// Initialize logger
	log.SetPrefix("[DuckDB MCP] ")
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Get database path from command line argument
	if len(os.Args) != 2 {
		log.Fatal("Usage: mcp-duckdb <path/to/database.duckdb>")
	}
	dbPath := os.Args[1]

	// Create and configure DuckDB tool
	tool := NewDuckDBTool(dbPath)
	defer tool.Close()

	// Get DuckDB version
	version, err := tool.GetVersion()
	if err != nil {
		log.Printf("Warning: Could not get DuckDB version: %v", err)
		version = "unknown"
	}
	// Clean up version string
	version = strings.TrimPrefix(version, "v")

	// Create server with stdio transport
	srv := server.NewServer(transport.NewStdioTransport())

	// Register the DuckDB tool
	if err := srv.RegisterTool(tool); err != nil {
		log.Fatalf("Failed to register DuckDB tool: %v", err)
	}

	// Start the server
	log.Printf("Starting DuckDB MCP server (DuckDB v%s) with database: %s", version, dbPath)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
