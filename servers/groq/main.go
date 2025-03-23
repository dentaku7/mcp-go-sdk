package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"mcp-go-sdk/server"
	"mcp-go-sdk/transport"
)

const (
	defaultModel       = "deepseek-r1-distill-llama-70b"
	defaultTemperature = 0.6
)

func main() {
	var config Config

	// Check environment variable first
	config.APIKey = os.Getenv("GROQ_API_KEY")

	// Define flags
	flag.StringVar(&config.APIKey, "api-key", config.APIKey, "Groq API key (can also be set via GROQ_API_KEY environment variable)")
	flag.StringVar(&config.Model, "model", "deepseek-r1-distill-llama-70b", "Model to use for completions")
	flag.Float64Var(&config.Temperature, "temperature", 0.6, "Temperature for response generation (0.0-1.5)")

	// Set up custom flag usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	// Validate required flags
	if config.APIKey == "" {
		log.Fatal("Error: API key is required. Set it via -api-key flag or GROQ_API_KEY environment variable")
	}

	// Validate temperature range
	if config.Temperature < 0 || config.Temperature > 1.5 {
		log.Fatal("Error: temperature must be between 0 and 1.5")
	}

	// Create a new MCP server with stdio transport
	srv := server.NewServer(transport.NewStdioTransport())

	// Create and register the Groq tool
	tool := NewGroqTool(config)
	if err := srv.RegisterTool(tool); err != nil {
		log.Fatalf("Failed to register tool: %v", err)
	}

	// Start the server
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
