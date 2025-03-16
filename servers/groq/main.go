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
	// Define flags
	var (
		apiKey      string
		model       string
		temperature float64
	)

	// Set up custom flag usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	// Define flags with descriptions
	flag.StringVar(&apiKey, "api-key", "", "Groq API key (required)")
	flag.StringVar(&model, "model", defaultModel, "Model to use for completions")
	flag.Float64Var(&temperature, "temperature", defaultTemperature, "Temperature for response generation (0.0-1.0)")
	flag.Parse()

	// Validate required flags
	if apiKey == "" {
		log.Fatal("Error: -api-key is required")
	}

	// Validate temperature range
	if temperature < 0 || temperature > 1 {
		log.Fatal("Error: temperature must be between 0 and 1")
	}

	// Create a new MCP server with stdio transport
	srv := server.NewServer(transport.NewStdioTransport())

	// Create and register the Groq tool
	tool := NewGroqTool(Config{
		Model:       model,
		Temperature: temperature,
		APIKey:      apiKey,
	})
	if err := srv.RegisterTool(tool); err != nil {
		log.Fatalf("Failed to register tool: %v", err)
	}

	// Start the server
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
