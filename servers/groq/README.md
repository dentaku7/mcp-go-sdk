# Groq Go Tool

A Go-based MCP tool that leverages the Groq API to provide intelligent responses and code assistance. This tool is designed to help developers by offering a second opinion on technical questions and validating solutions.

## Features

- Semantic understanding of technical questions
- Detailed explanations with step-by-step thinking process
- Code examples and best practices
- Integration with Groq's language models
- Support for temperature control to adjust response creativity
- MCP protocol compliance for IDE integration

## Prerequisites

- Go 1.23 or later
- A valid Groq API key

## Configuration

The tool accepts the following command-line arguments:

- `-api-key`: Groq API key (required)
- `-model`: Model to use for completions (default: "deepseek-r1-distill-llama-70b")
- `-temperature`: Temperature for response generation (0.0-1.0) (default: 0.6)

## Usage

### As a Command Line Tool

Run the tool directly:

```bash
groq -api-key "your-groq-api-key"
```

### As an MCP Tool

The tool implements the MCP protocol and can be integrated with Cursor IDE. To set it up:

1. Create a `.cursor/mcp.json` file in your project root:

```json
{
    "mcpServers": {
        "groq": {
            "command": "groq",
            "args": ["-api-key", "your-groq-api-key"],
            "env": {}
        }
    }
}
```

## Testing

Run the tests:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with a real API key
GROQ_API_KEY=gsk_*** go test -v ./...
```

## Development

The codebase follows standard Go practices and includes:

- Unit tests
- Integration tests
- Error handling
- Documentation
- MCP protocol implementation

## Error Handling

The tool includes robust error handling for:

- Invalid API keys
- Network issues
- Invalid parameters
- Model errors
- Protocol errors

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request 