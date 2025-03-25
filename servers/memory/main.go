package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mcp-go-sdk"
	"mcp-go-sdk/server"
	"mcp-go-sdk/transport"
	"mcp-memory/internal/graph"
	"mcp-memory/internal/tool"
)

// expandHomeDir replaces leading ~ with the user's home directory
func expandHomeDir(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %v", err)
	}

	// Replace ~ with home directory
	if len(path) > 1 {
		return filepath.Join(home, path[1:]), nil
	}
	return home, nil
}

// getDefaultMemoryPath returns the default path for the memory file
func getDefaultMemoryPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("error getting executable path: %v", err)
	}
	return filepath.Join(filepath.Dir(execPath), "memory.json"), nil
}

func main() {
	// Parse command-line flags
	var memoryPath string
	flag.StringVar(&memoryPath, "path", "", "Path to the memory file (required)")
	flag.Parse()

	// Trim any whitespace
	memoryPath = strings.TrimSpace(memoryPath)

	// Check environment variable if path not provided via flag
	if memoryPath == "" {
		memoryPath = strings.TrimSpace(os.Getenv("MEMORY_FILE_PATH"))
	}

	// Get default path if neither flag nor env var is set
	if memoryPath == "" {
		defaultPath, err := getDefaultMemoryPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting default memory path: %v\n", err)
			os.Exit(1)
		}
		memoryPath = defaultPath
	}

	// Validate that we have a non-empty path
	if strings.TrimSpace(memoryPath) == "" {
		fmt.Fprintf(os.Stderr, "Error: memory path cannot be empty. Provide a path using --path flag or MEMORY_FILE_PATH environment variable.\n")
		os.Exit(1)
	}

	// Expand home directory if path starts with ~
	expanded, err := expandHomeDir(memoryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error expanding home directory: %v\n", err)
		os.Exit(1)
	}
	memoryPath = expanded

	// If path is not absolute after expansion, make it relative to executable
	if !filepath.IsAbs(memoryPath) {
		execPath, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
			os.Exit(1)
		}
		memoryPath = filepath.Join(filepath.Dir(execPath), memoryPath)
	}

	// Create knowledge graph manager
	manager := graph.NewKnowledgeGraphManager(memoryPath)

	// Create a new server with stdin/stdout transport
	srv := server.NewServer(transport.NewStdioTransport())

	// Register all tools
	tools := []mcp.Tool{
		tool.NewCreateEntitiesTool(manager),
		tool.NewCreateRelationsTool(manager),
		tool.NewAddObservationsTool(manager),
		tool.NewDeleteEntitiesTool(manager),
		tool.NewDeleteObservationsTool(manager),
		tool.NewDeleteRelationsTool(manager),
		tool.NewReadGraphTool(manager),
		tool.NewSearchNodesTool(manager),
		tool.NewOpenNodesTool(manager),
	}

	for _, t := range tools {
		if err := srv.RegisterTool(t); err != nil {
			fmt.Fprintf(os.Stderr, "Error registering tool %s: %v\n", t.Name(), err)
			os.Exit(1)
		}
	}

	// Start the server
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}
